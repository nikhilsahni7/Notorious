package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"notorious-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// ── 1. User identity ──────────────────────────────────────────────────
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

	// ── 2. Search summary ─────────────────────────────────────────────────
	var totalSearches int
	var firstSearchAt, lastSearchAt *time.Time
	h.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), MIN(searched_at), MAX(searched_at)
		FROM search_history WHERE user_id = $1
	`, userID).Scan(&totalSearches, &firstSearchAt, &lastSearchAt)

	// avg searches per day = total / max(days since created_at, 1)
	daysSinceCreation := int(time.Since(identity.CreatedAt).Hours()/24) + 1
	avgSearchesPerDay := 0.0
	if daysSinceCreation > 0 {
		avgSearchesPerDay = float64(totalSearches) / float64(daysSinceCreation)
	}

	// ── 3. Top 10 search terms ────────────────────────────────────────────
	type TermFreq struct {
		Query string `json:"query"`
		Count int    `json:"count"`
	}
	topTerms := make([]TermFreq, 0, 10)
	rows, _ := h.db.Pool.Query(ctx, `
		SELECT query, COUNT(*) as freq
		FROM search_history
		WHERE user_id = $1
		GROUP BY query
		ORDER BY freq DESC
		LIMIT 10
	`, userID)
	if rows != nil {
		for rows.Next() {
			var t TermFreq
			rows.Scan(&t.Query, &t.Count)
			topTerms = append(topTerms, t)
		}
		rows.Close()
	}

	// ── 4. Daily volume – last 30 days ─────────────────────────────────────
	type DayVolume struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	dailyVolume := make([]DayVolume, 0, 30)
	rows, _ = h.db.Pool.Query(ctx, `
		SELECT DATE((searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::text as d, COUNT(*) as cnt
		FROM search_history
		WHERE user_id = $1
		  AND searched_at >= NOW() - INTERVAL '30 days'
		GROUP BY d
		ORDER BY d
	`, userID)
	if rows != nil {
		for rows.Next() {
			var dv DayVolume
			rows.Scan(&dv.Date, &dv.Count)
			dailyVolume = append(dailyVolume, dv)
		}
		rows.Close()
	}

	// ── 5. Peak hour ──────────────────────────────────────────────────────
	peakHour := -1
	peakHourFormatted := "N/A"
	if totalSearches > 0 {
		h.db.Pool.QueryRow(ctx, `
			SELECT EXTRACT(HOUR FROM (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::int
			FROM search_history
			WHERE user_id = $1
			GROUP BY 1
			ORDER BY COUNT(*) DESC
			LIMIT 1
		`, userID).Scan(&peakHour)
		if peakHour >= 0 {
			peakHourFormatted = formatHour(peakHour)
		}
	}

	// ── 6. Zero-result searches ───────────────────────────────────────────
	var zeroResultCount int
	h.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM search_history WHERE user_id = $1 AND total_results = 0
	`, userID).Scan(&zeroResultCount)

	zeroResultPct := 0.0
	if totalSearches > 0 {
		zeroResultPct = float64(zeroResultCount) / float64(totalSearches) * 100
	}

	// ── 7. Longest gap between searches (days) ───────────────────────────
	longestGapDays := 0
	if totalSearches > 1 {
		h.db.Pool.QueryRow(ctx, `
			SELECT COALESCE(MAX(gap_days), 0)::int FROM (
				SELECT EXTRACT(DAY FROM
					(searched_at - LAG(searched_at) OVER (ORDER BY searched_at))
				)::int as gap_days
				FROM search_history
				WHERE user_id = $1
			) t WHERE gap_days IS NOT NULL
		`, userID).Scan(&longestGapDays)
	}

	// ── 8. Password reset requests ────────────────────────────────────────
	var totalPasswordResets int
	var lastPasswordResetAt *time.Time
	h.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), MAX(created_at) FROM password_change_requests WHERE user_id = $1
	`, userID).Scan(&totalPasswordResets, &lastPasswordResetAt)

	// ── 9. Session / device count ─────────────────────────────────────────
	var devicesRegistered int
	h.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_sessions WHERE user_id = $1
	`, userID).Scan(&devicesRegistered)

	// ── 10. Last login (proxy: most recent session last_active) ──────────
	var lastLogin *time.Time
	h.db.Pool.QueryRow(ctx, `
		SELECT MAX(last_active) FROM user_sessions WHERE user_id = $1
	`, userID).Scan(&lastLogin)

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

	// ── 1. Search volume ──────────────────────────────────────────────────
	searchVolume := h.querySearchVolume(ctx)

	// ── 2. User patterns ──────────────────────────────────────────────────
	userPatterns := h.queryUserPatterns(ctx)

	// ── 3. Top search terms (system-wide) ────────────────────────────────
	type TermFreq struct {
		Query string `json:"query"`
		Count int    `json:"count"`
	}
	topTerms := make([]TermFreq, 0, 20)
	rows, _ := h.db.Pool.Query(ctx, `
		SELECT query, COUNT(*) as freq
		FROM search_history
		GROUP BY query
		ORDER BY freq DESC
		LIMIT 20
	`)
	if rows != nil {
		for rows.Next() {
			var t TermFreq
			rows.Scan(&t.Query, &t.Count)
			topTerms = append(topTerms, t)
		}
		rows.Close()
	}

	// ── 4. Zero results ───────────────────────────────────────────────────
	var totalSearchesForPct, zeroResultCount int
	h.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM search_history`).Scan(&totalSearchesForPct)
	h.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM search_history WHERE total_results = 0`).Scan(&zeroResultCount)
	zeroResultPct := 0.0
	if totalSearchesForPct > 0 {
		zeroResultPct = float64(zeroResultCount) / float64(totalSearchesForPct) * 100
	}

	// ── 5. Password resets ────────────────────────────────────────────────
	var totalPasswordResets, passwordResetsLast30d int
	h.db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days')
		FROM password_change_requests
	`).Scan(&totalPasswordResets, &passwordResetsLast30d)

	// ── 6. Users exceeding device limit ───────────────────────────────────
	type DeviceExceeded struct {
		ID           uuid.UUID `json:"id"`
		Name         string    `json:"name"`
		Email        string    `json:"email"`
		DeviceLimit  int       `json:"device_limit"`
		SessionCount int       `json:"session_count"`
	}
	usersExceedingLimit := make([]DeviceExceeded, 0)
	rows, _ = h.db.Pool.Query(ctx, `
		SELECT u.id, u.name, u.email, u.device_limit, COUNT(us.id) as session_count
		FROM users u
		JOIN user_sessions us ON u.id = us.user_id
		GROUP BY u.id, u.name, u.email, u.device_limit
		HAVING COUNT(us.id) > u.device_limit
		ORDER BY session_count DESC
		LIMIT 10
	`)
	if rows != nil {
		for rows.Next() {
			var d DeviceExceeded
			rows.Scan(&d.ID, &d.Name, &d.Email, &d.DeviceLimit, &d.SessionCount)
			usersExceedingLimit = append(usersExceedingLimit, d)
		}
		rows.Close()
	}

	// ── 7. Time distributions ─────────────────────────────────────────────
	timeDistributions := h.queryTimeDistributions(ctx)

	// ── Assemble response ─────────────────────────────────────────────────
	c.JSON(http.StatusOK, gin.H{
		"search_volume": searchVolume,
		"user_patterns": userPatterns,
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
		"time_distributions": timeDistributions,
	})
}

// querySearchVolume runs all search volume related queries.
func (h *StatsGinHandler) querySearchVolume(ctx context.Context) gin.H {
	var totalAllTime, totalLast30d int
	h.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM search_history`).Scan(&totalAllTime)
	h.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM search_history WHERE searched_at >= NOW() - INTERVAL '30 days'`).Scan(&totalLast30d)

	// avg daily = total / days since oldest search
	avgDaily := 0.0
	var oldestSearch *time.Time
	h.db.Pool.QueryRow(ctx, `SELECT MIN(searched_at) FROM search_history`).Scan(&oldestSearch)
	if oldestSearch != nil && totalAllTime > 0 {
		days := int(time.Since(*oldestSearch).Hours()/24) + 1
		avgDaily = float64(totalAllTime) / float64(days)
	}

	// avg per user per day
	var totalUsers int
	h.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'user'`).Scan(&totalUsers)
	avgPerUserPerDay := 0.0
	if totalUsers > 0 && avgDaily > 0 {
		avgPerUserPerDay = avgDaily / float64(totalUsers)
	}

	// daily trend last 90 days
	type DayVolume struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	dailyTrend := make([]DayVolume, 0, 90)
	rows, _ := h.db.Pool.Query(ctx, `
		SELECT DATE((searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::text as d, COUNT(*) as cnt
		FROM search_history
		WHERE searched_at >= NOW() - INTERVAL '90 days'
		GROUP BY d
		ORDER BY d
	`)
	if rows != nil {
		for rows.Next() {
			var dv DayVolume
			rows.Scan(&dv.Date, &dv.Count)
			dailyTrend = append(dailyTrend, dv)
		}
		rows.Close()
	}

	// peak system hour
	peakHour := -1
	peakHourFormatted := "N/A"
	if totalAllTime > 0 {
		h.db.Pool.QueryRow(ctx, `
			SELECT EXTRACT(HOUR FROM (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::int
			FROM search_history
			GROUP BY 1
			ORDER BY COUNT(*) DESC
			LIMIT 1
		`).Scan(&peakHour)
		if peakHour >= 0 {
			peakHourFormatted = formatHour(peakHour)
		}
	}

	return gin.H{
		"total_all_time":       totalAllTime,
		"total_last_30_days":   totalLast30d,
		"avg_daily":            fmt.Sprintf("%.1f", avgDaily),
		"avg_per_user_per_day": fmt.Sprintf("%.2f", avgPerUserPerDay),
		"daily_trend":          dailyTrend,
		"peak_hour":            peakHour,
		"peak_hour_formatted":  peakHourFormatted,
	}
}

// queryUserPatterns runs user engagement and activity queries.
func (h *StatsGinHandler) queryUserPatterns(ctx context.Context) gin.H {
	var totalUsers, activeUsersLast30d int
	h.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'user'`).Scan(&totalUsers)
	h.db.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM search_history
		WHERE searched_at >= NOW() - INTERVAL '30 days'
	`).Scan(&activeUsersLast30d)

	// avg searches per user (all time)
	avgSearchesPerUser := 0.0
	if totalUsers > 0 {
		var totalSearches int
		h.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM search_history`).Scan(&totalSearches)
		avgSearchesPerUser = float64(totalSearches) / float64(totalUsers)
	}

	// most active users (top 10)
	type ActiveUser struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Email       string    `json:"email"`
		SearchCount int       `json:"search_count"`
	}
	mostActive := make([]ActiveUser, 0, 10)
	rows, _ := h.db.Pool.Query(ctx, `
		SELECT u.id, u.name, u.email, COUNT(sh.id) as search_count
		FROM users u
		JOIN search_history sh ON u.id = sh.user_id
		GROUP BY u.id, u.name, u.email
		ORDER BY search_count DESC
		LIMIT 10
	`)
	if rows != nil {
		for rows.Next() {
			var au ActiveUser
			rows.Scan(&au.ID, &au.Name, &au.Email, &au.SearchCount)
			mostActive = append(mostActive, au)
		}
		rows.Close()
	}

	// device count distribution
	type DeviceBucket struct {
		DeviceCount int `json:"device_count"`
		UserCount   int `json:"user_count"`
	}
	deviceDistribution := make([]DeviceBucket, 0)
	rows, _ = h.db.Pool.Query(ctx, `
		SELECT device_count, COUNT(*) as user_count FROM (
			SELECT user_id, COUNT(*) as device_count
			FROM user_sessions
			GROUP BY user_id
		) t
		GROUP BY device_count
		ORDER BY device_count
	`)
	if rows != nil {
		for rows.Next() {
			var db DeviceBucket
			rows.Scan(&db.DeviceCount, &db.UserCount)
			deviceDistribution = append(deviceDistribution, db)
		}
		rows.Close()
	}

	return gin.H{
		"total_users":            totalUsers,
		"active_users_last_30d":  activeUsersLast30d,
		"avg_searches_per_user":  fmt.Sprintf("%.1f", avgSearchesPerUser),
		"most_active_users":      mostActive,
		"device_distribution":    deviceDistribution,
	}
}

// queryTimeDistributions runs hour/day-of-week/month distribution queries.
func (h *StatsGinHandler) queryTimeDistributions(ctx context.Context) gin.H {
	// by hour (0-23)
	type HourBucket struct {
		Hour  int    `json:"hour"`
		Label string `json:"label"`
		Count int    `json:"count"`
	}
	byHour := make([]HourBucket, 0, 24)
	rows, _ := h.db.Pool.Query(ctx, `
		SELECT EXTRACT(HOUR FROM (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::int as h, COUNT(*) as cnt
		FROM search_history
		GROUP BY h
		ORDER BY h
	`)
	if rows != nil {
		for rows.Next() {
			var hb HourBucket
			rows.Scan(&hb.Hour, &hb.Count)
			hb.Label = formatHour(hb.Hour)
			byHour = append(byHour, hb)
		}
		rows.Close()
	}

	// by day of week (0=Sunday … 6=Saturday)
	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	type DowBucket struct {
		Dow     int    `json:"dow"`
		DayName string `json:"day_name"`
		Count   int    `json:"count"`
	}
	byDow := make([]DowBucket, 0, 7)
	rows, _ = h.db.Pool.Query(ctx, `
		SELECT EXTRACT(DOW FROM (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')::int as d, COUNT(*) as cnt
		FROM search_history
		GROUP BY d
		ORDER BY d
	`)
	if rows != nil {
		for rows.Next() {
			var db DowBucket
			rows.Scan(&db.Dow, &db.Count)
			if db.Dow >= 0 && db.Dow < 7 {
				db.DayName = dayNames[db.Dow]
			}
			byDow = append(byDow, db)
		}
		rows.Close()
	}

	// by month (last 12 months)
	type MonthBucket struct {
		Month string `json:"month"`
		Count int    `json:"count"`
	}
	byMonth := make([]MonthBucket, 0, 12)
	rows, _ = h.db.Pool.Query(ctx, `
		SELECT TO_CHAR(DATE_TRUNC('month', (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata'), 'Mon YYYY') as m,
		       COUNT(*) as cnt
		FROM search_history
		WHERE searched_at >= NOW() - INTERVAL '12 months'
		GROUP BY DATE_TRUNC('month', (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')
		ORDER BY DATE_TRUNC('month', (searched_at AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Kolkata')
	`)
	if rows != nil {
		for rows.Next() {
			var mb MonthBucket
			rows.Scan(&mb.Month, &mb.Count)
			byMonth = append(byMonth, mb)
		}
		rows.Close()
	}

	return gin.H{
		"by_hour":        byHour,
		"by_day_of_week": byDow,
		"by_month":       byMonth,
	}
}
