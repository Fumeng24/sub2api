package repository

import (
	"database/sql"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const accountSchedulabilitySelectColumns = `
	id,
	platform,
	type,
	status,
	schedulable,
	expires_at,
	auto_pause_on_expired,
	rate_limit_reset_at,
	overload_until,
	temp_unschedulable_until,
	CASE
		-- Only API key / Bedrock quota fields affect SchedulingBlockReasonAt here.
		WHEN type IN ('apikey', 'bedrock') THEN COALESCE(extra, '{}'::jsonb)
		ELSE '{}'::jsonb
	END`

const joinedAccountSchedulabilitySelectColumns = `
	a.id,
	a.platform,
	a.type,
	a.status,
	a.schedulable,
	a.expires_at,
	a.auto_pause_on_expired,
	a.rate_limit_reset_at,
	a.overload_until,
	a.temp_unschedulable_until,
	CASE
		-- Only API key / Bedrock quota fields affect SchedulingBlockReasonAt here.
		WHEN a.type IN ('apikey', 'bedrock') THEN COALESCE(a.extra, '{}'::jsonb)
		ELSE '{}'::jsonb
	END`

type accountSchedulabilityScanTarget struct {
	account                *service.Account
	expiresAt              sql.NullTime
	rateLimitResetAt       sql.NullTime
	overloadUntil          sql.NullTime
	tempUnschedulableUntil sql.NullTime
	extraRaw               json.RawMessage
}

func newAccountSchedulabilityScanTarget(account *service.Account) *accountSchedulabilityScanTarget {
	return &accountSchedulabilityScanTarget{account: account}
}

func (t *accountSchedulabilityScanTarget) destinations() []any {
	return []any{
		&t.account.ID,
		&t.account.Platform,
		&t.account.Type,
		&t.account.Status,
		&t.account.Schedulable,
		&t.expiresAt,
		&t.account.AutoPauseOnExpired,
		&t.rateLimitResetAt,
		&t.overloadUntil,
		&t.tempUnschedulableUntil,
		&t.extraRaw,
	}
}

func (t *accountSchedulabilityScanTarget) apply() error {
	if t.expiresAt.Valid {
		t.account.ExpiresAt = &t.expiresAt.Time
	}
	if t.rateLimitResetAt.Valid {
		t.account.RateLimitResetAt = &t.rateLimitResetAt.Time
	}
	if t.overloadUntil.Valid {
		t.account.OverloadUntil = &t.overloadUntil.Time
	}
	if t.tempUnschedulableUntil.Valid {
		t.account.TempUnschedulableUntil = &t.tempUnschedulableUntil.Time
	}
	if len(t.extraRaw) > 0 {
		if err := json.Unmarshal(t.extraRaw, &t.account.Extra); err != nil {
			return err
		}
	}
	return nil
}

func scanAccountSchedulabilityRow(scan func(dest ...any) error, account *service.Account) error {
	target := newAccountSchedulabilityScanTarget(account)
	if err := scan(target.destinations()...); err != nil {
		return err
	}
	return target.apply()
}
