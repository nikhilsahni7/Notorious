# PERF_NOTES.md

Performance optimization log. Optimizations only — no behaviour changes.

## Not touched

Unrelated bugs / issues noticed during STEP 1 analysis (left as-is, per HARD RULES):

- `internal/handlers/admin_gin.go:333` — uses `println()` (builtin, writes to stderr) for error logging instead of `log`.
- `internal/handlers/admin_gin.go:322-339` (`ListUserRequests`) and `:435-454` (`GetSearchHistory`) — no `limit <= 0` guard; `?limit=0` (or negative) passes through to SQL as-is, unlike `ListUsers` which guards it. Pre-existing behaviour, preserved.
- `internal/handlers/admin_gin.go:420-425` (`RejectUserRequest`) — `GetByID` error ignored and fields set on `userRequest` are never persisted (dead code).
- `internal/handlers/admin_gin.go:885` (`GenerateUserEOD`) — CSV "Search ID" is derived from the slice index (`searchID+1`), not the row's real ID; fragile but intentional-looking.
- `internal/handlers/stats_gin.go` — all query errors ignored (zero values surface in JSON on DB failure); error-handling style, not touched.
- `internal/handlers/search.go:359` (`ExportEODReport`?) — search history created synchronously with request context (no timeout); inconsistent with the async pattern at `search.go:220-224`.
- `internal/middleware/rate_limiter.go`, `bot_blocker.go` — not reviewed in depth this pass.
- `gofmt -l` flags several pre-existing files unformatted (`auth/jwt.go`, `database/postgres.go`, `handlers/stats_gin.go`, `handlers/upload.go`, `handlers/user_password_gin.go`, `models/chat.go`, `utils/ip_utils.go`, `websocket/hub.go`, `main.go`) — the misalignments are in regions unrelated to the optimizations; left as-is per HARD RULES.

## Optimizations

### 1. GetSystemStats: dedupe repeated queries → 19 → 15 queries/request
`internal/handlers/stats_gin.go:249-262` (and `querySearchVolume` / `queryUserPatterns` signatures)

- **What changed:** `SELECT COUNT(*) FROM search_history` was run **3× per request** (`GetSystemStats` zero-pct block, `querySearchVolume`, `queryUserPatterns`) and `SELECT COUNT(*) FROM users WHERE role = 'user'` **2×**. Each is now run once in `GetSystemStats` and passed into the helpers; `COUNT(*)` + `MIN(searched_at)` merged into one statement.
- **Why faster:** removes 4 sequential DB round trips and 2 redundant full-table `COUNT(*)` scans (~243k rows each in the local DB). No index scan vs seq scan change — pure dedupe + one fewer column pass.
- **Before/after:** within noise for wall-clock (1.003s → 1.007s, p=0.912) because the remaining cost is 7 independent full-table GROUP BY scans (see next entry). B/op: 52.22Ki → 42.83Ki. Goldens byte-identical (`system_stats.json` compared, not rewritten).
- **Behaviour risk:** none on success paths; on a failed query the old code zero-filled only that metric, the merged statements zero-fill their metric group — same JSON for healthy DB. Checked via golden comparison with `-race`.

### 2. GetSystemStats: run independent aggregations concurrently → −47.9% wall-clock
`internal/handlers/stats_gin.go:263-371`

- **What changed:** the 7 independent blocks (search volume, user patterns, top terms, zero-results, password resets, device-limit violators, time distributions) now run in an `errgroup` instead of sequentially. Queries themselves unchanged.
- **Why faster:** the endpoint was a sequential chain of full-table GROUP BY/hash-agg scans over `search_history` (~243k rows each, ~100-200ms apiece). Wall-clock was the **sum**; it is now the **max** of one block. Pool (25 conns) absorbs ≤9 concurrent queries.
- **Before/after:** `Handler_GetSystemStats` 1002.7ms ± 3% → **522.3ms ± 2%** (−47.9%, p=0.000, n=10; `benchstat old.txt new.txt`).
- **Behaviour risk:** none on success (goldens byte-identical, `-race` clean). Query errors still zero-fill their metric (same semantics). Deviation only in the pathological panic path: previously a panic in a section was caught by `gin.Recovery` (plain 500); now each block is recovered with the existing `recoverDashboardStatsPanic` pattern and returns 500 `{"error": "failed to load system stats"}` — matching the dashboard-stats precedent in this file.

### 3. GetUserStats: merge aggregates + concurrent blocks → −24.4% wall-clock
`internal/handlers/stats_gin.go:80-236`

- **What changed:** `COUNT/MIN/MAX` and the zero-result count merged into one statement (`COUNT(*) FILTER (WHERE total_results = 0)` — same value, one scan instead of two); `COUNT(*)` + `MAX(last_active)` on `user_sessions` merged into one; the 5 independent metric blocks (summary, top terms, daily volume, password resets, sessions) now run concurrently. Peak-hour and longest-gap blocks stay sequential and keep their original `totalSearches > 0` / `> 1` gates, which are known only after the merged summary returns.
- **Why faster:** 10 sequential round trips → 5 concurrent + 2 gated sequential (7 total); two scans of the user's `search_history` rows and two of the user's `user_sessions` rows eliminated outright.
- **Before/after:** `Handler_GetUserStats` 1.369ms ± 2% → **1.036ms ± 1%** (−24.4%, p=0.000, n=10, vs the STEP 2 baseline `old.txt`). Scales with the user's row count (prolific users gain more).
- **Behaviour risk:** none on success (goldens byte-identical incl. `user_stats.json`, `-race` clean; error → zero-fill unchanged, identity error → same 404). Panic path deviation as in entry 2 (`failed to load user stats` 500).

### 4. GetConversations: correlated COUNT subquery → single aggregated join
`internal/repository/chat_repository.go:145-186`

- **What changed:** the per-row `(SELECT COUNT(*) … WHERE conversation_id = c.id AND receiver_id = $1 AND status != 'read')` subselect is replaced by one `LEFT JOIN` against a pre-aggregated set of the user's unread messages, `GROUP BY conversation_id`, hitting `idx_chat_messages_receiver_unread (receiver_id, status) WHERE status != 'read'` once.
- **Why faster:** was O(conversations × messages) — an index probe per conversation row (also executed on every WebSocket connect via `hub.go:201`). Now one index scan over the user's unread messages regardless of conversation count.
- **Before/after:** local admin has no conversations (`[]`), so the local bench can only prove no regression: 105.7µs ± 4% → ~103µs, allocs unchanged (104). Real gain requires data; expected queries-per-request unchanged (1) but index probes: conversations → 1.
- **Behaviour risk:** low. Column order, `COALESCE(…, 0)` semantics and `ORDER BY last_message_at DESC` preserved; the original COALESCE never produced anything but 0 (COUNT over empty set), the new LEFT JOIN genuinely can be NULL, hence the COALESCE is load-bearing here. Goldens cover only the empty set (local DB has no chat rows) — verified by reasoning on SQL equivalence, not by data. Rows with *identical* `last_message_at` ties could theoretically reorder (both plans allow arbitrary tie order); timestamps come from `NOW()` updates so collisions are unlikely.

### 5. Migrations 012-014: login email expression index + status/created_at composite indexes
`migrations/012_users_email_lower_trim_index.sql`, `013_user_requests_status_created_at.sql`, `014_password_change_requests_status_created_at.sql`

- **What changed:** no code; three `CREATE INDEX CONCURRENTLY` statement files (one statement each — CONCURRENTLY cannot run inside the implicit transaction the migration runner uses for multi-statement Exec).
- **Why faster:** `WHERE lower(trim(email)) = lower(trim($1))` (login + revoke-session) currently seq-scans `users` — verified locally: `Rows Removed by Filter: 98`. The list endpoints `WHERE status = $1 ORDER BY created_at DESC` sort every matching row per page. Index scan + no top-N sort after apply.
- **Before/after:** pending apply (HARD RULE: I don't run DDL). Run `EXPLAIN (ANALYZE, BUFFERS)` before —
  `EXPLAIN (ANALYZE, BUFFERS) SELECT id FROM users WHERE lower(trim(email)) = lower(trim('x@example.com'));` (already captured: Seq Scan) — and again after the server restart applies them (expect Index Scan via `idx_users_email_lower_trim`; expect no explicit Sort node for `user_requests`/`password_change_requests` listings).
- **Behaviour risk:** none — indexes only, no data or planner-visible output change.

### 6. GenerateUserEOD: CSV escaping — one allocation per cell instead of O(n²) copies
`internal/handlers/admin_gin.go:844-878`

- **What changed:** `escapeCSV`'s quoted-cell builder replaced the char-by-char `escaped += …` loop with a pre-grown `strings.Builder` (`b.Grow(len(value)+2)`). Output bytes identical: same `!`→`,` preprocessing, same quote-doubling order, same wrapping condition.
- **Why faster:** Go strings are immutable — every `+=` allocated a fresh copy of everything built so far (quadratic bytes-copied for a long value). One buffer, one allocation per quoted cell.
- **Before/after:** endpoint not benchmarked (low-frequency CSV export); gain is bounded and small in absolute terms — this is hygiene, not a hot-path win. Golden CSV comparison isn't in the golden set (binary/CSV, not JSON); verified by mechanical equivalence of the loop logic.
- **Behaviour risk:** none — same characters emitted in the same order.

### 7. Review round (independent clean-code audit of the full diff)
- **Applied from review:** `rows.Close()` → `defer rows.Close()` in all 10 row-iteration loops in `stats_gin.go` that execute inside the new recovered blocks — an unrecovered-panic path would previously have permanently abandoned a pooled connection; now the pool connection is always returned.
- **Applied from review (hygiene):** import placement + `var`-block alignment for the newly added regions of `stats_gin.go` (`gofmt` on pre-existing misaligned blocks was **not** run — those regions predate this work and are untouched); `go mod tidy` promoted `golang.org/x/sync` from indirect to direct in `go.mod`.
- **Verdict:** "Approve with nits". Verified empirically by the reviewer: `-race` golden run, goldens captured before the rewrites still match byte-for-byte, errgroup rewrites have no shared-variable races, merged queries value-identical, chat rewrite column-order/COALESCE/ordering identical, migrations match the real query shapes, test env skips cleanly when DATABASE_URL is unset.
- **Known deviations (documented):** panic-path 500 body is now JSON `{"error": "failed to load … stats"}` instead of `gin.Recovery`'s plain 500 body; status code identical.
