package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"notorious-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

// StatsGinHandler handles analytics endpoints for per-user and system-wide stats.
// It operates directly against the DB pool for cross-table aggregation queries.
type StatsGinHandler struct {
	db *database.DB
}

func NewStatsGinHandler(db *database.DB) *StatsGinHandler {
	return &StatsGinHandler{db: db}
}

// formatHour converts 0-23 integer hour to "12 AM", "6 PM" format.
func formatHour(hour int) string {
	if hour == 0 {
		return "12 AM"
	}
	if hour == 12 {
		return "12 PM"
	}
	if hour < 12 {
		return fmt.Sprintf("%d AM", hour)
	}
	return fmt.Sprintf("%d PM", hour-12)
}

// ─────────────────────────────────────────────
// GET /api/admin/stats/user/:userId
// ─────────────────────────────────────────────

func (h *StatsGinHandler) GetUserStats(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()

	// ── 1. User identity (must succeed before parallel queries) ───────────
	type UserIdentity struct {
		ID               uuid.UUID `json:"id"`
		Email            string    `json:"email"`
		Name             string    `json:"name"`
		CreatedAt        time.Time `json:"created_at"`
		IsActive         bool      `json:"is_active"`
		DeviceLimit      int       `json:"device_limit"`
		DailySearchLimit int       `json:"daily_search_limit"`
		SearchesUsed     int       `json:"searches_used_today"`
		Region           string    `json:"region"`
	}

	var identity UserIdentity
	err = h.db.Pool.QueryRow(ctx, `
		SELECT id, email, name, created_at, is_active, device_limit,
		       daily_search_limit, searches_used_today,
		       COALESCE(region, 'pan-india') as region
		FROM users WHERE id = $1
	`, userID).Scan(
		&identity.ID, &identity.Email, &identity.Name, &identity.CreatedAt,
		&identity.IsActive, &identity.DeviceLimit, &identity.DailySearchLimit,
		&identity.SearchesUsed, &identity.Region,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// ── Run all remaining queries in parallel ─────────────────────────────
	var (
		totalSearches                int
		firstSearchAt, lastSearchAt  *time.Time
		zeroResultCount              int
		longestGapDays               int
		totalPasswordResets          int
		lastPasswordResetAt          *time.Time
		devicesRegistered            int
		lastLogin                    *time.Time
		peakHour                     int = -1
	)

	type TermFreq struct {
		Query string `json:"query"`
		Count int    `json:"count"`
	}
	type DayVolume struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	topTerms := make([]TermFreq, 0, 10)
	dailyVolume := make([]DayVolume, 0, 30)

	g, gctx := errgroup.WithContext(ctx)

	// 2. Search summary
	g.Go(func() error {
		return h.db.Pool.QueryRow(gctx, `
			SELECT COUNT(*), MIN(searched_at), MAX(searched_at)
			FROM search_history WHERE user_id = $1
		`, userID).Scan(&totalSearches, &firstSearchAt, &lastSearchAt)
	})

	// 3. Top 10 search terms
	g.Go(func() error {
		rows, err := h.db.Pool.Query(gctx, `
			SELECT query, COUNT(*) as freq
			FROM search_history
			WHERE user_id = $1
			GROUP BY query
			ORDER BY freq DESC
			LIMIT 10
		`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var t TermFreq
			if err := rows.Scan(&t.Query, &t.Count); err != nil {
				return err
			}
			topTerms = append(topTerms, t)
		}
		return rows.Err()
	})

	// 4. Daily volume – last 30 days
	g.Go(func() error {
		rows, err := h.db.Pool.Query(gctx, `
			SELECT DATE((searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::text as d, COUNT(*) as cnt
			FROM search_history
			WHERE user_id = $1
			  AND searched_at >= NOW() - INTERVAL '30 days'
			GROUP BY d
			ORDER BY d
		`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var dv DayVolume
			if err := rows.Scan(&dv.Date, &dv.Count); err != nil {
				return err
			}
			dailyVolume = append(dailyVolume, dv)
		}
		return rows.Err()
	})

	// 5. Zero-result searches
	g.Go(func() error {
		return h.db.Pool.QueryRow(gctx, `
			SELECT COUNT(*) FROM search_history WHERE user_id = $1 AND total_results = 0
		`, userID).Scan(&zeroResultCount)
	})

	// 6. Longest gap
	g.Go(func() error {
		err := h.db.Pool.QueryRow(gctx, `
			SELECT COALESCE(MAX(gap_days), 0)::int FROM (
				SELECT EXTRACT(DAY FROM
					(searched_at - LAG(searched_at) OVER (ORDER BY searched_at))
				)::int as gap_days
				FROM search_history
				WHERE user_id = $1
			) t WHERE gap_days IS NOT NULL
		`, userID).Scan(&longestGapDays)
		if err == pgx.ErrNoRows {
			return nil // no data is not an error
		}
		return err
	})

	// 7. Password reset requests
	g.Go(func() error {
		return h.db.Pool.QueryRow(gctx, `
			SELECT COUNT(*), MAX(created_at) FROM password_change_requests WHERE user_id = $1
		`, userID).Scan(&totalPasswordResets, &lastPasswordResetAt)
	})

	// 8. Session / device count + last login
	g.Go(func() error {
		return h.db.Pool.QueryRow(gctx, `
			SELECT COUNT(*), MAX(last_active) FROM user_sessions WHERE user_id = $1
		`, userID).Scan(&devicesRegistered, &lastLogin)
	})

	// 9. Peak hour (ErrNoRows if user has no searches — not an error)
	g.Go(func() error {
		err := h.db.Pool.QueryRow(gctx, `
			SELECT EXTRACT(HOUR FROM (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::int
			FROM search_history
			WHERE user_id = $1
			GROUP BY 1
			ORDER BY COUNT(*) DESC
			LIMIT 1
		`, userID).Scan(&peakHour)
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	})

	if err := g.Wait(); err != nil {
		log.Printf("stats: user %s query error: %v", userID, err)
		// Non-fatal: continue with whatever data we have
	}

	// ── Compute derived values ────────────────────────────────────────────
	daysSinceCreation := int(time.Since(identity.CreatedAt).Hours()/24) + 1
	avgSearchesPerDay := 0.0
	if daysSinceCreation > 0 {
		avgSearchesPerDay = float64(totalSearches) / float64(daysSinceCreation)
	}

	zeroResultPct := 0.0
	if totalSearches > 0 {
		zeroResultPct = float64(zeroResultCount) / float64(totalSearches) * 100
	}

	peakHourFormatted := "N/A"
	if peakHour >= 0 {
		peakHourFormatted = formatHour(peakHour)
	}

	// ── Assemble response ─────────────────────────────────────────────────
	c.JSON(http.StatusOK, gin.H{
		"identity": gin.H{
			"id":                 identity.ID,
			"email":              identity.Email,
			"name":               identity.Name,
			"created_at":         identity.CreatedAt,
			"is_active":          identity.IsActive,
			"device_limit":       identity.DeviceLimit,
			"daily_search_limit": identity.DailySearchLimit,
			"searches_used_today": identity.SearchesUsed,
			"region":             identity.Region,
		},
		"sessions": gin.H{
			"devices_registered": devicesRegistered,
			"last_login":         lastLogin,
		},
		"search_behavior": gin.H{
			"total_searches":      totalSearches,
			"avg_searches_per_day": fmt.Sprintf("%.2f", avgSearchesPerDay),
			"first_search_at":     firstSearchAt,
			"last_search_at":      lastSearchAt,
			"top_terms":           topTerms,
			"daily_volume":        dailyVolume,
			"peak_hour":           peakHour,
			"peak_hour_formatted": peakHourFormatted,
			"zero_result_searches": zeroResultCount,
			"zero_result_pct":     fmt.Sprintf("%.1f", zeroResultPct),
		},
		"security": gin.H{
			"total_password_reset_requests": totalPasswordResets,
			"last_password_reset_at":        lastPasswordResetAt,
			"devices_registered":            devicesRegistered,
			"device_limit":                  identity.DeviceLimit,
		},
		"engagement": gin.H{
			"first_search_at":  firstSearchAt,
			"last_search_at":   lastSearchAt,
			"longest_gap_days": longestGapDays,
		},
	})
}

// ─────────────────────────────────────────────
// GET /api/admin/stats/system
// ─────────────────────────────────────────────

func (h *StatsGinHandler) GetSystemStats(c *gin.Context) {
	ctx := c.Request.Context()

	// All independent query groups run in parallel.
	var (
		// search volume
		totalAllTime    int
		totalLast30d    int
		oldestSearch    *time.Time
		totalUsers      int
		peakHour        int = -1

		// zero result
		zeroResultCount int

		// password resets
		totalPasswordResets    int
		passwordResetsLast30d  int
	)

	type TermFreq struct {
		Query string `json:"query"`
		Count int    `json:"count"`
	}
	type DayVolume struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	type ActiveUser struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Email       string    `json:"email"`
		SearchCount int       `json:"search_count"`
	}
	type DeviceExceeded struct {
		ID           uuid.UUID `json:"id"`
		Name         string    `json:"name"`
		Email        string    `json:"email"`
		DeviceLimit  int       `json:"device_limit"`
		SessionCount int       `json:"session_count"`
	}
	type DeviceBucket struct {
		DeviceCount int `json:"device_count"`
		UserCount   int `json:"user_count"`
	}
	type HourBucket struct {
		Hour  int    `json:"hour"`
		Label string `json:"label"`
		Count int    `json:"count"`
	}
	type DowBucket struct {
		Dow     int    `json:"dow"`
		DayName string `json:"day_name"`
		Count   int    `json:"count"`
	}
	type MonthBucket struct {
		Month string `json:"month"`
		Count int    `json:"count"`
	}

	topTerms := make([]TermFreq, 0, 20)
	dailyTrend := make([]DayVolume, 0, 90)
	mostActive := make([]ActiveUser, 0, 10)
	usersExceedingLimit := make([]DeviceExceeded, 0)
	deviceDistribution := make([]DeviceBucket, 0)
	activeUsersLast30d := 0
	avgSearchesPerUser := 0.0

	byHour := make([]HourBucket, 0, 24)
	byDow := make([]DowBucket, 0, 7)
	byMonth := make([]MonthBucket, 0, 12)

	g, gctx := errgroup.WithContext(ctx)

	// ── 1. Core counts (totalAllTime, totalLast30d, oldest, zeroResults, totalUsers) ──
	// These used to be 4 separate SELECT COUNT(*) FROM search_history queries.
	// Now combined into a single query that computes all counts in one pass.
	g.Go(func() error {
		return h.db.Pool.QueryRow(gctx, `
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE searched_at >= NOW() - INTERVAL '30 days'),
				COUNT(*) FILTER (WHERE total_results = 0),
				MIN(searched_at)
			FROM search_history
		`).Scan(&totalAllTime, &totalLast30d, &zeroResultCount, &oldestSearch)
	})

	g.Go(func() error {
		return h.db.Pool.QueryRow(gctx, `
			SELECT COUNT(*) FROM users WHERE role = 'user'
		`).Scan(&totalUsers)
	})

	// ── 2. Top search terms ───────────────────────────────────────────────
	g.Go(func() error {
		rows, err := h.db.Pool.Query(gctx, `
			SELECT query, COUNT(*) as freq
			FROM search_history
			GROUP BY query
			ORDER BY freq DESC
			LIMIT 20
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var t TermFreq
			if err := rows.Scan(&t.Query, &t.Count); err != nil {
				return err
			}
			topTerms = append(topTerms, t)
		}
		return rows.Err()
	})

	// ── 3. Daily trend last 90 days ───────────────────────────────────────
	g.Go(func() error {
		rows, err := h.db.Pool.Query(gctx, `
			SELECT DATE((searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::text as d, COUNT(*) as cnt
			FROM search_history
			WHERE searched_at >= NOW() - INTERVAL '90 days'
			GROUP BY d
			ORDER BY d
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var dv DayVolume
			if err := rows.Scan(&dv.Date, &dv.Count); err != nil {
				return err
			}
			dailyTrend = append(dailyTrend, dv)
		}
		return rows.Err()
	})

	// ── 4. Peak system hour (ErrNoRows if no searches exist — not an error)
	g.Go(func() error {
		err := h.db.Pool.QueryRow(gctx, `
			SELECT EXTRACT(HOUR FROM (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::int
			FROM search_history
			GROUP BY 1
			ORDER BY COUNT(*) DESC
			LIMIT 1
		`).Scan(&peakHour)
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	})

	// ── 5. Password resets ────────────────────────────────────────────────
	g.Go(func() error {
		return h.db.Pool.QueryRow(gctx, `
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days')
			FROM password_change_requests
		`).Scan(&totalPasswordResets, &passwordResetsLast30d)
	})

	// ── 6. Users exceeding device limit ───────────────────────────────────
	g.Go(func() error {
		rows, err := h.db.Pool.Query(gctx, `
			SELECT u.id, u.name, u.email, u.device_limit, COUNT(us.id) as session_count
			FROM users u
			JOIN user_sessions us ON u.id = us.user_id
			GROUP BY u.id, u.name, u.email, u.device_limit
			HAVING COUNT(us.id) > u.device_limit
			ORDER BY session_count DESC
			LIMIT 10
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var d DeviceExceeded
			if err := rows.Scan(&d.ID, &d.Name, &d.Email, &d.DeviceLimit, &d.SessionCount); err != nil {
				return err
			}
			usersExceedingLimit = append(usersExceedingLimit, d)
		}
		return rows.Err()
	})

	// ── 7. Active users last 30d + most active users ──────────────────────
	g.Go(func() error {
		if err := h.db.Pool.QueryRow(gctx, `
			SELECT COUNT(DISTINCT user_id) FROM search_history
			WHERE searched_at >= NOW() - INTERVAL '30 days'
		`).Scan(&activeUsersLast30d); err != nil {
			return err
		}
		rows, err := h.db.Pool.Query(gctx, `
			SELECT u.id, u.name, u.email, COUNT(sh.id) as search_count
			FROM users u
			JOIN search_history sh ON u.id = sh.user_id
			GROUP BY u.id, u.name, u.email
			ORDER BY search_count DESC
			LIMIT 10
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var au ActiveUser
			if err := rows.Scan(&au.ID, &au.Name, &au.Email, &au.SearchCount); err != nil {
				return err
			}
			mostActive = append(mostActive, au)
		}
		return rows.Err()
	})

	// ── 8. Device distribution ────────────────────────────────────────────
	g.Go(func() error {
		rows, err := h.db.Pool.Query(gctx, `
			SELECT device_count, COUNT(*) as user_count FROM (
				SELECT user_id, COUNT(*) as device_count
				FROM user_sessions
				GROUP BY user_id
			) t
			GROUP BY device_count
			ORDER BY device_count
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var db DeviceBucket
			if err := rows.Scan(&db.DeviceCount, &db.UserCount); err != nil {
				return err
			}
			deviceDistribution = append(deviceDistribution, db)
		}
		return rows.Err()
	})

	// ── 9. Time distributions (hour, dow, month) ──────────────────────────
	g.Go(func() error {
		rows, err := h.db.Pool.Query(gctx, `
			SELECT EXTRACT(HOUR FROM (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::int as h, COUNT(*) as cnt
			FROM search_history
			GROUP BY h
			ORDER BY h
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var hb HourBucket
			if err := rows.Scan(&hb.Hour, &hb.Count); err != nil {
				return err
			}
			hb.Label = formatHour(hb.Hour)
			byHour = append(byHour, hb)
		}
		return rows.Err()
	})

	g.Go(func() error {
		dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		rows, err := h.db.Pool.Query(gctx, `
			SELECT EXTRACT(DOW FROM (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::int as d, COUNT(*) as cnt
			FROM search_history
			GROUP BY d
			ORDER BY d
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var db DowBucket
			if err := rows.Scan(&db.Dow, &db.Count); err != nil {
				return err
			}
			if db.Dow >= 0 && db.Dow < 7 {
				db.DayName = dayNames[db.Dow]
			}
			byDow = append(byDow, db)
		}
		return rows.Err()
	})

	g.Go(func() error {
		rows, err := h.db.Pool.Query(gctx, `
			SELECT TO_CHAR(DATE_TRUNC('month', (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata'), 'Mon YYYY') as m,
			       COUNT(*) as cnt
			FROM search_history
			WHERE searched_at >= NOW() - INTERVAL '12 months'
			GROUP BY DATE_TRUNC('month', (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')
			ORDER BY DATE_TRUNC('month', (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var mb MonthBucket
			if err := rows.Scan(&mb.Month, &mb.Count); err != nil {
				return err
			}
			byMonth = append(byMonth, mb)
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		log.Printf("stats: system query error: %v", err)
		// Non-fatal: continue with whatever data we have
	}

	// ── Compute derived values ────────────────────────────────────────────
	avgDaily := 0.0
	if oldestSearch != nil && totalAllTime > 0 {
		days := int(time.Since(*oldestSearch).Hours()/24) + 1
		avgDaily = float64(totalAllTime) / float64(days)
	}

	avgPerUserPerDay := 0.0
	if totalUsers > 0 && avgDaily > 0 {
		avgPerUserPerDay = avgDaily / float64(totalUsers)
	}

	if totalUsers > 0 {
		avgSearchesPerUser = float64(totalAllTime) / float64(totalUsers)
	}

	zeroResultPct := 0.0
	if totalAllTime > 0 {
		zeroResultPct = float64(zeroResultCount) / float64(totalAllTime) * 100
	}

	peakHourFormatted := "N/A"
	if peakHour >= 0 {
		peakHourFormatted = formatHour(peakHour)
	}

	// ── Assemble response ─────────────────────────────────────────────────
	c.JSON(http.StatusOK, gin.H{
		"search_volume": gin.H{
			"total_all_time":       totalAllTime,
			"total_last_30_days":   totalLast30d,
			"avg_daily":            fmt.Sprintf("%.1f", avgDaily),
			"avg_per_user_per_day": fmt.Sprintf("%.2f", avgPerUserPerDay),
			"daily_trend":          dailyTrend,
			"peak_hour":            peakHour,
			"peak_hour_formatted":  peakHourFormatted,
		},
		"user_patterns": gin.H{
			"total_users":            totalUsers,
			"active_users_last_30d":  activeUsersLast30d,
			"avg_searches_per_user":  fmt.Sprintf("%.1f", avgSearchesPerUser),
			"most_active_users":      mostActive,
			"device_distribution":    deviceDistribution,
		},
		"search_patterns": gin.H{
			"top_terms":         topTerms,
			"zero_result_count": zeroResultCount,
			"zero_result_pct":   fmt.Sprintf("%.1f", zeroResultPct),
		},
		"security": gin.H{
			"total_password_resets":          totalPasswordResets,
			"password_resets_last_30_days":   passwordResetsLast30d,
			"users_exceeding_device_limit":   usersExceedingLimit,
		},
		"time_distributions": gin.H{
			"by_hour":        byHour,
			"by_day_of_week": byDow,
			"by_month":       byMonth,
		},
	})
}
