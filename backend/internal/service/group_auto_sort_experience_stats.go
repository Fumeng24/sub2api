package service

import (
	"context"
	"database/sql"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

type groupAutoSortModelExperienceStats struct {
	SuccessCount  int64
	FailureCount  int64
	FailoverCount int64
}

type groupAutoSortExperienceStats struct {
	SuccessCount        int64
	FailureCount        int64
	FailoverCount       int64
	FirstTokenSamples   int64
	DurationSamples     int64
	P95FirstTokenMs     float64
	P95DurationMs       float64
	CacheReadTokens     int64
	CacheEligibleTokens int64
	Models              map[string]*groupAutoSortModelExperienceStats
}

func (s *groupAutoSortExperienceStats) attempts() int64 {
	if s == nil {
		return 0
	}
	return s.SuccessCount + s.FailureCount
}

func (s *groupAutoSortExperienceStats) failureRate() float64 {
	if attempts := s.attempts(); attempts > 0 {
		return float64(s.FailureCount) / float64(attempts)
	}
	return 0
}

func (s *groupAutoSortExperienceStats) failoverRate() float64 {
	if attempts := s.attempts(); attempts > 0 {
		return float64(s.FailoverCount) / float64(attempts)
	}
	return 0
}

func (s *groupAutoSortExperienceStats) cacheHitRate() float64 {
	if s == nil || s.CacheEligibleTokens <= 0 {
		return 0
	}
	return float64(s.CacheReadTokens) / float64(s.CacheEligibleTokens)
}

func (s *groupAutoSortExperienceStats) worstModelFailureRate(minAttempts int64) float64 {
	if s == nil {
		return 0
	}
	worst := 0.0
	for _, model := range s.Models {
		if model == nil {
			continue
		}
		attempts := model.SuccessCount + model.FailureCount
		if attempts < minAttempts {
			continue
		}
		rate := float64(model.FailureCount) / float64(attempts)
		if rate > worst {
			worst = rate
		}
	}
	return worst
}

type groupAutoSortExperienceProvider interface {
	StatsByAccountID(
		ctx context.Context,
		groupID int64,
		accountIDs []int64,
		since time.Time,
	) (map[int64]*groupAutoSortExperienceStats, error)
}

// groupAutoSortWeightedExperienceProvider is optional so lightweight test and
// third-party providers can keep implementing the original interface. The SQL
// provider uses a 30-minute recent window blended with a 24-hour decayed
// baseline.
type groupAutoSortWeightedExperienceProvider interface {
	StatsByAccountIDWeighted(
		ctx context.Context,
		groupID int64,
		accountIDs []int64,
		recentSince time.Time,
		longSince time.Time,
	) (map[int64]*groupAutoSortExperienceStats, error)
}

type groupAutoSortRateProvider interface {
	RatesByAccountID(ctx context.Context, accountIDs []int64, now time.Time) (map[int64]float64, error)
}

type sqlGroupAutoSortExperienceProvider struct {
	db *sql.DB
}

func (p *sqlGroupAutoSortExperienceProvider) StatsByAccountIDWeighted(
	ctx context.Context,
	groupID int64,
	accountIDs []int64,
	recentSince time.Time,
	longSince time.Time,
) (map[int64]*groupAutoSortExperienceStats, error) {
	if p == nil || p.db == nil {
		return map[int64]*groupAutoSortExperienceStats{}, nil
	}
	recent, err := p.StatsByAccountID(ctx, groupID, accountIDs, recentSince)
	if err != nil {
		return nil, err
	}
	long, err := p.StatsByAccountID(ctx, groupID, accountIDs, longSince)
	if err != nil {
		return nil, err
	}
	return blendGroupAutoSortExperienceStats(recent, long), nil
}

// blendGroupAutoSortExperienceStats gives recent traffic 70% influence and
// the 24-hour baseline 30%. Counts are intentionally blended rather than
// concatenated because the long window includes the recent window.
func blendGroupAutoSortExperienceStats(recent, long map[int64]*groupAutoSortExperienceStats) map[int64]*groupAutoSortExperienceStats {
	out := make(map[int64]*groupAutoSortExperienceStats, len(recent)+len(long))
	ids := make(map[int64]struct{}, len(recent)+len(long))
	for id := range recent {
		ids[id] = struct{}{}
	}
	for id := range long {
		ids[id] = struct{}{}
	}
	for id := range ids {
		r := recent[id]
		l := long[id]
		if r == nil {
			r = &groupAutoSortExperienceStats{}
		}
		if l == nil {
			l = &groupAutoSortExperienceStats{}
		}
		v := &groupAutoSortExperienceStats{Models: make(map[string]*groupAutoSortModelExperienceStats)}
		v.SuccessCount = weightedCount(r.SuccessCount, l.SuccessCount)
		v.FailureCount = weightedCount(r.FailureCount, l.FailureCount)
		v.FailoverCount = weightedCount(r.FailoverCount, l.FailoverCount)
		v.FirstTokenSamples = weightedCount(r.FirstTokenSamples, l.FirstTokenSamples)
		v.DurationSamples = weightedCount(r.DurationSamples, l.DurationSamples)
		v.P95FirstTokenMs = weightedFloat(r.P95FirstTokenMs, l.P95FirstTokenMs, r.FirstTokenSamples, l.FirstTokenSamples)
		v.P95DurationMs = weightedFloat(r.P95DurationMs, l.P95DurationMs, r.DurationSamples, l.DurationSamples)
		v.CacheReadTokens = weightedCount(r.CacheReadTokens, l.CacheReadTokens)
		v.CacheEligibleTokens = weightedCount(r.CacheEligibleTokens, l.CacheEligibleTokens)
		modelNames := make(map[string]struct{}, len(r.Models)+len(l.Models))
		for name := range r.Models {
			modelNames[name] = struct{}{}
		}
		for name := range l.Models {
			modelNames[name] = struct{}{}
		}
		for name := range modelNames {
			rm, lm := r.Models[name], l.Models[name]
			if rm == nil {
				rm = &groupAutoSortModelExperienceStats{}
			}
			if lm == nil {
				lm = &groupAutoSortModelExperienceStats{}
			}
			v.Models[name] = &groupAutoSortModelExperienceStats{
				SuccessCount:  weightedCount(rm.SuccessCount, lm.SuccessCount),
				FailureCount:  weightedCount(rm.FailureCount, lm.FailureCount),
				FailoverCount: weightedCount(rm.FailoverCount, lm.FailoverCount),
			}
		}
		out[id] = v
	}
	return out
}

func weightedCount(recent, long int64) int64 {
	return int64(math.Round(float64(recent)*0.70 + float64(long)*0.30))
}

func weightedFloat(recent, long float64, recentSamples, longSamples int64) float64 {
	if recentSamples <= 0 {
		return long
	}
	if longSamples <= 0 {
		return recent
	}
	return recent*0.70 + long*0.30
}

func newSQLGroupAutoSortExperienceProvider(db *sql.DB) *sqlGroupAutoSortExperienceProvider {
	if db == nil {
		return nil
	}
	return &sqlGroupAutoSortExperienceProvider{db: db}
}

func (p *sqlGroupAutoSortExperienceProvider) RatesByAccountID(
	ctx context.Context,
	accountIDs []int64,
	now time.Time,
) (map[int64]float64, error) {
	rates := make(map[int64]float64, len(accountIDs))
	if p == nil || p.db == nil || len(accountIDs) == 0 {
		return rates, nil
	}
	const query = `
		WITH bound AS (
			SELECT
				a.id AS account_id,
				u.metadata->'account_billing'->(a.id::text) AS billing,
				u.metadata->>'management_status' AS management_status
			FROM accounts a
			JOIN upstreams u ON u.id = a.upstream_id
			WHERE a.id = ANY($1)
				AND a.deleted_at IS NULL
				AND u.deleted_at IS NULL
		)
		SELECT
			account_id,
			billing->>'group_effective_rate_multiplier' AS effective_rate,
			billing->>'group_default_rate_multiplier' AS default_rate,
			billing->>'fetched_at' AS fetched_at
		FROM bound
		WHERE management_status = 'ok'
			AND billing->>'status' = 'ok'
			AND COALESCE((billing->>'stale')::boolean, FALSE) = FALSE
	`
	rows, err := p.db.QueryContext(ctx, query, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			accountID    int64
			effectiveRaw sql.NullString
			defaultRaw   sql.NullString
			fetchedRaw   sql.NullString
		)
		if err := rows.Scan(&accountID, &effectiveRaw, &defaultRaw, &fetchedRaw); err != nil {
			return nil, err
		}
		if !fetchedRaw.Valid {
			continue
		}
		fetchedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(fetchedRaw.String))
		if err != nil || fetchedAt.IsZero() || now.Before(fetchedAt) || now.Sub(fetchedAt) > groupAutoSortUpstreamRateFreshness {
			continue
		}
		raw := effectiveRaw
		if !raw.Valid || strings.TrimSpace(raw.String) == "" {
			raw = defaultRaw
		}
		if !raw.Valid {
			continue
		}
		rate, err := strconv.ParseFloat(strings.TrimSpace(raw.String), 64)
		if err != nil {
			continue
		}
		if rate, ok := validGroupAutoSortRate(rate); ok {
			rates[accountID] = rate
		}
	}
	return rates, rows.Err()
}

func (p *sqlGroupAutoSortExperienceProvider) StatsByAccountID(
	ctx context.Context,
	groupID int64,
	accountIDs []int64,
	since time.Time,
) (map[int64]*groupAutoSortExperienceStats, error) {
	stats := make(map[int64]*groupAutoSortExperienceStats, len(accountIDs))
	if p == nil || p.db == nil || groupID <= 0 || len(accountIDs) == 0 {
		return stats, nil
	}
	if err := p.loadSuccessStats(ctx, stats, groupID, accountIDs, since); err != nil {
		return nil, err
	}
	if err := p.loadFailureStats(ctx, stats, groupID, accountIDs, since); err != nil {
		return nil, err
	}
	return stats, nil
}

func ensureGroupAutoSortExperienceStats(stats map[int64]*groupAutoSortExperienceStats, accountID int64) *groupAutoSortExperienceStats {
	value := stats[accountID]
	if value == nil {
		value = &groupAutoSortExperienceStats{Models: make(map[string]*groupAutoSortModelExperienceStats)}
		stats[accountID] = value
	}
	if value.Models == nil {
		value.Models = make(map[string]*groupAutoSortModelExperienceStats)
	}
	return value
}

func ensureGroupAutoSortModelExperienceStats(stats *groupAutoSortExperienceStats, model string) *groupAutoSortModelExperienceStats {
	model = strings.TrimSpace(model)
	value := stats.Models[model]
	if value == nil {
		value = &groupAutoSortModelExperienceStats{}
		stats.Models[model] = value
	}
	return value
}

func (p *sqlGroupAutoSortExperienceProvider) loadSuccessStats(
	ctx context.Context,
	stats map[int64]*groupAutoSortExperienceStats,
	groupID int64,
	accountIDs []int64,
	since time.Time,
) error {
	const query = `
		WITH samples AS (
			SELECT
				account_id,
				COALESCE(NULLIF(requested_model, ''), NULLIF(model, ''), 'unknown') AS model_name,
				first_token_ms,
				duration_ms,
				input_tokens,
				cache_read_tokens
			FROM usage_logs ul
			WHERE ul.group_id = $1
				AND ul.account_id = ANY($2)
				AND ul.created_at >= $3
				AND NOT EXISTS (
					SELECT 1
					FROM channel_monitors cm
					WHERE cm.api_key_id = ul.api_key_id
						AND (
							COALESCE(ul.user_agent, '') LIKE 'Go-http-client/%'
							OR EXISTS (
								SELECT 1
								FROM jsonb_each_text(COALESCE(cm.extra_headers, '{}'::jsonb))
									AS monitor_header(header_name, header_value)
								WHERE LOWER(monitor_header.header_name) = 'user-agent'
									AND monitor_header.header_value = COALESCE(ul.user_agent, '')
							)
						)
				)
		)
		SELECT
			account_id,
			CASE WHEN GROUPING(model_name) = 1 THEN NULL ELSE model_name END AS model_name,
			COUNT(*)::bigint AS success_count,
			COUNT(first_token_ms)::bigint AS first_token_samples,
			COUNT(duration_ms)::bigint AS duration_samples,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY first_token_ms)
				FILTER (WHERE first_token_ms IS NOT NULL)::float8 AS p95_first_token_ms,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms)
				FILTER (WHERE duration_ms IS NOT NULL)::float8 AS p95_duration_ms,
			COALESCE(SUM(cache_read_tokens), 0)::bigint AS cache_read_tokens,
			COALESCE(SUM(input_tokens + cache_read_tokens), 0)::bigint AS cache_eligible_tokens
		FROM samples
		GROUP BY GROUPING SETS ((account_id), (account_id, model_name))
	`
	rows, err := p.db.QueryContext(ctx, query, groupID, pq.Array(accountIDs), since)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			accountID           int64
			modelName           sql.NullString
			successCount        int64
			firstTokenCount     int64
			durationCount       int64
			p95FirstToken       sql.NullFloat64
			p95Duration         sql.NullFloat64
			cacheReadTokens     int64
			cacheEligibleTokens int64
		)
		if err := rows.Scan(
			&accountID,
			&modelName,
			&successCount,
			&firstTokenCount,
			&durationCount,
			&p95FirstToken,
			&p95Duration,
			&cacheReadTokens,
			&cacheEligibleTokens,
		); err != nil {
			return err
		}
		accountStats := ensureGroupAutoSortExperienceStats(stats, accountID)
		if modelName.Valid {
			ensureGroupAutoSortModelExperienceStats(accountStats, modelName.String).SuccessCount = successCount
			continue
		}
		accountStats.SuccessCount = successCount
		accountStats.FirstTokenSamples = firstTokenCount
		accountStats.DurationSamples = durationCount
		accountStats.CacheReadTokens = cacheReadTokens
		accountStats.CacheEligibleTokens = cacheEligibleTokens
		if p95FirstToken.Valid {
			accountStats.P95FirstTokenMs = p95FirstToken.Float64
		}
		if p95Duration.Valid {
			accountStats.P95DurationMs = p95Duration.Float64
		}
	}
	return rows.Err()
}

func (p *sqlGroupAutoSortExperienceProvider) loadFailureStats(
	ctx context.Context,
	stats map[int64]*groupAutoSortExperienceStats,
	groupID int64,
	accountIDs []int64,
	since time.Time,
) error {
	const query = `
		WITH scoped AS (
			SELECT
				e.id,
				COALESCE(NULLIF(e.request_id, ''), NULLIF(e.client_request_id, ''), e.id::text) AS request_key,
				account_id,
				COALESCE(NULLIF(requested_model, ''), NULLIF(model, ''), 'unknown') AS model_name,
				status_code,
				error_owner,
				COALESCE(NULLIF(upstream_errors, 'null'::jsonb), '[]'::jsonb) AS upstream_errors
			FROM ops_error_logs e
			WHERE e.group_id = $1
				AND e.created_at >= $3
				AND e.is_count_tokens = FALSE
				AND NOT EXISTS (
					SELECT 1
					FROM channel_monitors cm
					WHERE cm.api_key_id = e.api_key_id
						AND (
							COALESCE(e.user_agent, '') LIKE 'Go-http-client/%'
							OR EXISTS (
								SELECT 1
								FROM jsonb_each_text(COALESCE(cm.extra_headers, '{}'::jsonb))
									AS monitor_header(header_name, header_value)
								WHERE LOWER(monitor_header.header_name) = 'user-agent'
									AND monitor_header.header_value = COALESCE(e.user_agent, '')
							)
						)
				)
		), event_rows AS (
			SELECT
				(ev->>'account_id')::bigint AS account_id,
				s.model_name,
				split_part(COALESCE(ev->>'kind', ''), ':', 1) AS kind,
				CASE
					WHEN COALESCE(ev->>'upstream_status_code', '') ~ '^[0-9]{3}$'
					THEN (ev->>'upstream_status_code')::int
					ELSE 0
				END AS upstream_status
			FROM scoped s
			CROSS JOIN LATERAL jsonb_array_elements(s.upstream_errors) AS ev
			WHERE COALESCE(ev->>'account_id', '') ~ '^[0-9]+$'
				AND (ev->>'account_id')::bigint = ANY($2)
		), significant_events AS (
			SELECT account_id, model_name, kind
			FROM event_rows
			WHERE kind IN (
				'failover', 'retry_exhausted_failover', 'failover_on_400',
				'credential_failover', 'request_error', 'retry_exhausted'
			)
			OR upstream_status IN (401, 402, 403, 408, 429, 502, 503, 504)
			OR upstream_status >= 500
		), final_provider_error_rows AS (
			SELECT account_id, model_name, request_key,
				ROW_NUMBER() OVER (PARTITION BY request_key, account_id, model_name ORDER BY id DESC) AS row_number
			FROM scoped
			WHERE account_id = ANY($2)
				AND error_owner = 'provider'
				AND status_code >= 400
		), final_provider_errors AS (
			SELECT account_id, model_name, 'final_provider_error'::text AS kind
			FROM final_provider_error_rows
			WHERE row_number = 1
		), classified_events AS (
			-- A request contributes at most one visible failure, regardless of
			-- how many upstream attempts failed. Failover attempts are tracked
			-- separately so recovered requests do not look like user-visible
			-- errors while still influencing the stability score lightly.
			SELECT account_id, model_name, 1::bigint AS failure_count, 0::bigint AS failover_count
			FROM final_provider_errors
			UNION ALL
			SELECT account_id, model_name, 0::bigint AS failure_count, 1::bigint AS failover_count
			FROM significant_events
			WHERE kind IN ('failover', 'retry_exhausted_failover', 'failover_on_400', 'credential_failover')
		)
		SELECT
			account_id,
			CASE WHEN GROUPING(model_name) = 1 THEN NULL ELSE model_name END AS model_name,
			COALESCE(SUM(failure_count), 0)::bigint AS failure_count,
			COALESCE(SUM(failover_count), 0)::bigint AS failover_count
		FROM classified_events
		GROUP BY GROUPING SETS ((account_id), (account_id, model_name))
	`
	rows, err := p.db.QueryContext(ctx, query, groupID, pq.Array(accountIDs), since)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			accountID     int64
			modelName     sql.NullString
			failureCount  int64
			failoverCount int64
		)
		if err := rows.Scan(&accountID, &modelName, &failureCount, &failoverCount); err != nil {
			return err
		}
		accountStats := ensureGroupAutoSortExperienceStats(stats, accountID)
		if modelName.Valid {
			modelStats := ensureGroupAutoSortModelExperienceStats(accountStats, modelName.String)
			modelStats.FailureCount = failureCount
			modelStats.FailoverCount = failoverCount
			continue
		}
		accountStats.FailureCount = failureCount
		accountStats.FailoverCount = failoverCount
	}
	return rows.Err()
}
