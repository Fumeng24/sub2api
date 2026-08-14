package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ListByGroup and the helpers below are the local scheduler-history view. The
// upstream outbox reader remains unchanged; this overlay only adds the admin
// history projection and its presentation-time compaction.
func (r *schedulerOutboxRepository) ListByGroup(ctx context.Context, groupID int64, limit int) ([]service.SchedulerOutboxEvent, error) {
	if groupID <= 0 {
		return []service.SchedulerOutboxEvent{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	events, err := r.listByGroupFromTable(ctx, "scheduler_history", groupID, schedulerHistoryCandidateLimit(limit))
	if err == nil {
		return compactSchedulerHistoryEvents(events, limit), nil
	}
	if !isUndefinedTableError(err) {
		return nil, err
	}
	return r.listByGroupFromTable(ctx, "scheduler_outbox", groupID, limit)
}

func (r *schedulerOutboxRepository) listByGroupFromTable(ctx context.Context, table string, groupID int64, limit int) ([]service.SchedulerOutboxEvent, error) {
	query := fmt.Sprintf(`
		SELECT id, event_type, account_id, group_id, payload, created_at
		FROM %s
		WHERE group_id = $1
			OR account_id IN (
				SELECT account_id
				FROM account_groups
				WHERE group_id = $1
			)
			OR EXISTS (
				SELECT 1
				FROM jsonb_array_elements_text(COALESCE(payload->'group_ids', '[]'::jsonb)) AS gid(value)
				WHERE gid.value ~ '^[0-9]+$' AND gid.value::bigint = $1
			)
			OR EXISTS (
				SELECT 1
				FROM jsonb_array_elements_text(COALESCE(payload->'account_ids', '[]'::jsonb)) AS aid(value)
				JOIN account_groups ag ON ag.account_id = aid.value::bigint
				WHERE aid.value ~ '^[0-9]+$' AND ag.group_id = $1
			)
			OR EXISTS (
				SELECT 1
				FROM jsonb_object_keys(COALESCE(payload->'last_used', '{}'::jsonb)) AS aid(value)
				JOIN account_groups ag ON ag.account_id = aid.value::bigint
				WHERE aid.value ~ '^[0-9]+$' AND ag.group_id = $1
			)
		ORDER BY id DESC
		LIMIT $2
	`, table)
	rows, err := r.db.QueryContext(ctx, query, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	events := make([]service.SchedulerOutboxEvent, 0, limit)
	for rows.Next() {
		var (
			payloadRaw []byte
			accountID  sql.NullInt64
			groupIDVal sql.NullInt64
			event      service.SchedulerOutboxEvent
		)
		if err := rows.Scan(&event.ID, &event.EventType, &accountID, &groupIDVal, &payloadRaw, &event.CreatedAt); err != nil {
			return nil, err
		}
		if accountID.Valid {
			v := accountID.Int64
			event.AccountID = &v
		}
		if groupIDVal.Valid {
			v := groupIDVal.Int64
			event.GroupID = &v
		}
		if len(payloadRaw) > 0 {
			var payload map[string]any
			if err := json.Unmarshal(payloadRaw, &payload); err != nil {
				return nil, err
			}
			event.Payload = payload
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func compactSchedulerHistoryEvents(events []service.SchedulerOutboxEvent, limit int) []service.SchedulerOutboxEvent {
	if limit <= 0 {
		limit = 20
	}
	events = suppressNoisySchedulerAccountChanges(events)
	out := make([]service.SchedulerOutboxEvent, 0, minInt(limit, len(events)))
	seen := make(map[string]int, len(events))
	for _, event := range events {
		key := schedulerHistoryCompactKey(event)
		if idx, ok := seen[key]; ok {
			mergeSchedulerHistoryEvent(&out[idx], event)
			continue
		}
		if len(out) >= limit {
			continue
		}
		event.Payload = cloneSchedulerPayload(event.Payload)
		seen[key] = len(out)
		out = append(out, event)
	}
	return out
}

func schedulerHistoryCandidateLimit(limit int) int {
	if limit <= 0 {
		limit = 20
	}
	candidateLimit := limit * 20
	if candidateLimit < 200 {
		return 200
	}
	if candidateLimit > 1000 {
		return 1000
	}
	return candidateLimit
}

func suppressNoisySchedulerAccountChanges(events []service.SchedulerOutboxEvent) []service.SchedulerOutboxEvent {
	if len(events) == 0 {
		return events
	}
	filtered := make([]service.SchedulerOutboxEvent, 0, len(events))
	for _, event := range events {
		if isLowSignalSchedulerHistoryEvent(event) {
			continue
		}
		filtered = append(filtered, event)
	}
	events = filtered
	blockedAtByAccount := make(map[int64][]time.Time)
	for _, event := range events {
		if event.AccountID == nil {
			continue
		}
		switch event.EventType {
		case service.SchedulerOutboxEventSchedulingBlocked, service.SchedulerOutboxEventSchedulingBlockSkipped:
			blockedAtByAccount[*event.AccountID] = append(blockedAtByAccount[*event.AccountID], event.CreatedAt)
		}
	}
	if len(blockedAtByAccount) == 0 {
		return events
	}
	out := make([]service.SchedulerOutboxEvent, 0, len(events))
	for _, event := range events {
		if isNoisySchedulerAccountChange(event, blockedAtByAccount) {
			continue
		}
		out = append(out, event)
	}
	return out
}

func isLowSignalSchedulerHistoryEvent(event service.SchedulerOutboxEvent) bool {
	switch event.EventType {
	case service.SchedulerOutboxEventAccountLastUsed:
		return true
	case service.SchedulerOutboxEventAccountBulkChanged:
		return payloadOnlyHasAccountIDs(event.Payload)
	default:
		return false
	}
}

func payloadOnlyHasAccountIDs(payload map[string]any) bool {
	if len(payload) != 1 {
		return false
	}
	_, ok := payload["account_ids"]
	return ok
}

func isNoisySchedulerAccountChange(event service.SchedulerOutboxEvent, blockedAtByAccount map[int64][]time.Time) bool {
	if event.EventType != service.SchedulerOutboxEventAccountChanged || len(event.Payload) > 0 {
		return false
	}
	for _, accountID := range schedulerHistoryEventAccountIDs(event) {
		for _, blockedAt := range blockedAtByAccount[accountID] {
			diff := blockedAt.Sub(event.CreatedAt)
			if diff < 0 {
				diff = -diff
			}
			if diff <= 2*time.Second {
				return true
			}
		}
	}
	return false
}

func schedulerHistoryEventAccountIDs(event service.SchedulerOutboxEvent) []int64 {
	if event.AccountID != nil {
		return []int64{*event.AccountID}
	}
	if event.EventType != service.SchedulerOutboxEventAccountBulkChanged || event.Payload == nil {
		return nil
	}
	raw, ok := event.Payload["account_ids"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			if id, ok := numericInt64(item); ok && id > 0 {
				out = append(out, id)
			}
		}
		return out
	case []int64:
		return v
	case []int:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			if item > 0 {
				out = append(out, int64(item))
			}
		}
		return out
	default:
		if id, ok := numericInt64(v); ok && id > 0 {
			return []int64{id}
		}
		return nil
	}
}

func numericInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), v == float64(int64(v))
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

func schedulerHistoryCompactKey(event service.SchedulerOutboxEvent) string {
	if event.EventType != service.SchedulerOutboxEventSchedulingBlocked && event.EventType != service.SchedulerOutboxEventSchedulingBlockSkipped {
		return strings.Join([]string{event.EventType, strconv.FormatInt(event.ID, 10)}, "\x00")
	}
	payload := event.Payload
	parts := []string{event.EventType, int64PtrKey(event.AccountID), int64PtrKey(event.GroupID), payloadString(payload, "reason"), payloadString(payload, "source"), payloadString(payload, "failure_category"), payloadString(payload, "block_granularity"), payloadString(payload, "model_rate_limit"), payloadString(payload, "model")}
	return strings.Join(parts, "\x00")
}

func mergeSchedulerHistoryEvent(current *service.SchedulerOutboxEvent, older service.SchedulerOutboxEvent) {
	if current == nil {
		return
	}
	if current.Payload == nil {
		current.Payload = map[string]any{}
	}
	count := intFromPayload(current.Payload, "history_count")
	if count <= 0 {
		count = 1
	}
	current.Payload["history_count"] = count + 1
	current.Payload["history_first_at"] = older.CreatedAt.Format(time.RFC3339)
	if _, ok := current.Payload["history_last_at"]; !ok {
		current.Payload["history_last_at"] = current.CreatedAt.Format(time.RFC3339)
	}
}

func cloneSchedulerPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	clone := make(map[string]any, len(payload))
	for key, value := range payload {
		clone[key] = value
	}
	return clone
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	if v, ok := value.(fmt.Stringer); ok {
		return v.String()
	}
	return fmt.Sprint(value)
}

func intFromPayload(payload map[string]any, key string) int {
	if payload == nil {
		return 0
	}
	switch v := payload[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

func int64PtrKey(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// enqueueSchedulerOutboxCustom preserves the local history side effect while
// leaving the upstream insert and dedup query in scheduler_outbox_repo.go.
func enqueueSchedulerOutboxCustom(ctx context.Context, exec sqlExecutor, result sql.Result, outboxErr error, eventType string, accountID *int64, groupID *int64, payloadArg any) error {
	if outboxErr != nil {
		return outboxErr
	}
	if schedulerOutboxEventSupportsDedup(eventType) {
		rows, err := result.RowsAffected()
		if err == nil && rows == 0 {
			return nil
		}
	}
	if err := enqueueSchedulerHistory(ctx, exec, eventType, accountID, groupID, payloadArg); err != nil {
		logger.LegacyPrintf("repository.scheduler_outbox", "[SchedulerHistory] enqueue failed: event=%s err=%v", eventType, err)
	}
	return nil
}

func enqueueSchedulerHistory(ctx context.Context, exec sqlExecutor, eventType string, accountID *int64, groupID *int64, payloadArg any) error {
	query := `
		INSERT INTO scheduler_history (event_type, account_id, group_id, payload)
		VALUES ($1, $2, $3, $4)
	`
	if schedulerHistoryEventSupportsCooldownDedup(eventType) {
		query = `
			INSERT INTO scheduler_history (event_type, account_id, group_id, payload)
			SELECT $1, $2, $3, $4
			WHERE NOT EXISTS (
				SELECT 1
				FROM scheduler_history
				WHERE event_type = $1
					AND account_id IS NOT DISTINCT FROM $2
					AND group_id IS NOT DISTINCT FROM $3
					AND COALESCE(payload->>'reason', '') = COALESCE(($4::jsonb)->>'reason', '')
					AND COALESCE(payload->>'source', '') = COALESCE(($4::jsonb)->>'source', '')
					AND COALESCE(payload->>'failure_category', '') = COALESCE(($4::jsonb)->>'failure_category', '')
					AND COALESCE(payload->>'block_granularity', '') = COALESCE(($4::jsonb)->>'block_granularity', '')
					AND COALESCE(payload->>'model_rate_limit', '') = COALESCE(($4::jsonb)->>'model_rate_limit', '')
					AND COALESCE(payload->>'model', '') = COALESCE(($4::jsonb)->>'model', '')
					AND (
						created_at > NOW() - INTERVAL '5 minutes'
						OR CASE
							WHEN COALESCE(payload->>'until', '') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T'
							THEN (payload->>'until')::timestamptz > NOW()
							ELSE FALSE
						END
					)
				LIMIT 1
			)
		`
	}
	_, err := exec.ExecContext(ctx, query, eventType, accountID, groupID, payloadArg)
	if isUndefinedTableError(err) {
		return nil
	}
	return err
}

func schedulerHistoryEventSupportsCooldownDedup(eventType string) bool {
	switch eventType {
	case service.SchedulerOutboxEventSchedulingBlocked, service.SchedulerOutboxEventSchedulingBlockSkipped:
		return true
	default:
		return false
	}
}
