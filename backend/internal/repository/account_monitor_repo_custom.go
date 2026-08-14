package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/ent/accountmonitor"
	"github.com/Wei-Shaw/sub2api/ent/accountmonitorcheck"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// accountMonitorRepository 实现 service.AccountMonitorRepository。
// CRUD 走 ent；聚合查询（latest / 7d availability）走原生 SQL 命中索引。
type accountMonitorRepository struct {
	client *dbent.Client
	db     *sql.DB
}

// NewAccountMonitorRepository 创建仓储实例。
func NewAccountMonitorRepository(client *dbent.Client, db *sql.DB) service.AccountMonitorRepository {
	return &accountMonitorRepository{client: client, db: db}
}

// ---------- CRUD ----------

func (r *accountMonitorRepository) Create(ctx context.Context, m *service.AccountMonitor) error {
	client := clientFromContext(ctx, r.client)
	created, err := client.AccountMonitor.Create().
		SetAccountID(m.AccountID).
		SetProvider(accountmonitor.Provider(m.Provider)).
		SetModel(m.Model).
		SetEnabled(m.Enabled).
		SetIntervalSeconds(m.IntervalSeconds).
		SetJitterSeconds(m.JitterSeconds).
		SetCreatedBy(m.CreatedBy).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrAccountMonitorNotFound, service.ErrAccountMonitorExists)
	}
	m.ID = created.ID
	m.Provider = string(created.Provider)
	m.CreatedAt = created.CreatedAt
	m.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *accountMonitorRepository) Update(ctx context.Context, m *service.AccountMonitor) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.AccountMonitor.UpdateOneID(m.ID).
		SetProvider(accountmonitor.Provider(m.Provider)).
		SetModel(m.Model).
		SetEnabled(m.Enabled).
		SetIntervalSeconds(m.IntervalSeconds).
		SetJitterSeconds(m.JitterSeconds).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrAccountMonitorNotFound, nil)
	}
	return nil
}

func (r *accountMonitorRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	if err := client.AccountMonitor.DeleteOneID(id).Exec(ctx); err != nil {
		return translatePersistenceError(err, service.ErrAccountMonitorNotFound, nil)
	}
	return nil
}

func (r *accountMonitorRepository) GetByID(ctx context.Context, id int64) (*service.AccountMonitor, error) {
	row, err := r.client.AccountMonitor.Query().
		Where(accountmonitor.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAccountMonitorNotFound, nil)
	}
	return entToServiceAccountMonitor(row), nil
}

func (r *accountMonitorRepository) GetByAccountID(ctx context.Context, accountID int64) (*service.AccountMonitor, error) {
	row, err := r.client.AccountMonitor.Query().
		Where(accountmonitor.AccountIDEQ(accountID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAccountMonitorNotFound, nil)
	}
	return entToServiceAccountMonitor(row), nil
}

func (r *accountMonitorRepository) List(ctx context.Context) ([]*service.AccountMonitor, error) {
	rows, err := r.client.AccountMonitor.Query().
		Order(dbent.Asc(accountmonitor.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account monitors: %w", err)
	}
	return entsToServiceAccountMonitors(rows), nil
}

func (r *accountMonitorRepository) ListEnabled(ctx context.Context) ([]*service.AccountMonitor, error) {
	rows, err := r.client.AccountMonitor.Query().
		Where(accountmonitor.EnabledEQ(true)).
		Where(accountmonitor.HasAccountWith(dbaccount.DeletedAtIsNil())).
		Order(dbent.Asc(accountmonitor.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled account monitors: %w", err)
	}
	return entsToServiceAccountMonitors(rows), nil
}

func (r *accountMonitorRepository) UpdateLastCheckedAt(ctx context.Context, id int64, at time.Time) error {
	client := clientFromContext(ctx, r.client)
	if err := client.AccountMonitor.UpdateOneID(id).SetLastCheckedAt(at).Exec(ctx); err != nil {
		return translatePersistenceError(err, service.ErrAccountMonitorNotFound, nil)
	}
	return nil
}

// ---------- checks ----------

func (r *accountMonitorRepository) InsertChecks(ctx context.Context, checks []*service.AccountMonitorCheck) error {
	if len(checks) == 0 {
		return nil
	}
	client := clientFromContext(ctx, r.client)
	builders := make([]*dbent.AccountMonitorCheckCreate, 0, len(checks))
	for _, c := range checks {
		b := client.AccountMonitorCheck.Create().
			SetAccountMonitorID(c.AccountMonitorID).
			SetModel(c.Model).
			SetStatus(accountmonitorcheck.Status(c.Status)).
			SetMessage(c.Message).
			SetCheckedAt(c.CheckedAt)
		if c.LatencyMs != nil {
			b = b.SetLatencyMs(*c.LatencyMs)
		}
		if c.PingLatencyMs != nil {
			b = b.SetPingLatencyMs(*c.PingLatencyMs)
		}
		builders = append(builders, b)
	}
	if _, err := client.AccountMonitorCheck.CreateBulk(builders...).Save(ctx); err != nil {
		return fmt.Errorf("insert account monitor checks: %w", err)
	}
	return nil
}

// LatestChecks 取每个 monitorID 最近一条探测（DISTINCT ON 命中 (account_monitor_id, checked_at DESC) 索引）。
func (r *accountMonitorRepository) LatestChecks(ctx context.Context, monitorIDs []int64) (map[int64]*service.AccountMonitorCheck, error) {
	out := make(map[int64]*service.AccountMonitorCheck, len(monitorIDs))
	if len(monitorIDs) == 0 {
		return out, nil
	}
	const q = `
		SELECT DISTINCT ON (account_monitor_id)
		       account_monitor_id, model, status, latency_ms, ping_latency_ms, message, checked_at
		FROM account_monitor_checks
		WHERE account_monitor_id = ANY($1)
		ORDER BY account_monitor_id, checked_at DESC`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(monitorIDs))
	if err != nil {
		return nil, fmt.Errorf("query latest account monitor checks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			c       service.AccountMonitorCheck
			latency sql.NullInt64
			ping    sql.NullInt64
		)
		if err := rows.Scan(&c.AccountMonitorID, &c.Model, &c.Status, &latency, &ping, &c.Message, &c.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan latest account monitor check: %w", err)
		}
		if latency.Valid {
			v := int(latency.Int64)
			c.LatencyMs = &v
		}
		if ping.Valid {
			v := int(ping.Int64)
			c.PingLatencyMs = &v
		}
		cp := c
		out[c.AccountMonitorID] = &cp
	}
	return out, rows.Err()
}

// RecentChecks 取每个 monitorID 最近 limit 条探测（newest-first），供前端彩虹条渲染。
// 用窗口函数按 monitor 分组排序后截断，命中 (account_monitor_id, checked_at DESC) 索引。
func (r *accountMonitorRepository) RecentChecks(ctx context.Context, monitorIDs []int64, limit int) (map[int64][]*service.AccountMonitorCheck, error) {
	out := make(map[int64][]*service.AccountMonitorCheck, len(monitorIDs))
	if len(monitorIDs) == 0 || limit <= 0 {
		return out, nil
	}
	const q = `
		SELECT account_monitor_id, model, status, latency_ms, ping_latency_ms, message, checked_at
		FROM (
			SELECT account_monitor_id, model, status, latency_ms, ping_latency_ms, message, checked_at,
			       ROW_NUMBER() OVER (PARTITION BY account_monitor_id ORDER BY checked_at DESC) AS rn
			FROM account_monitor_checks
			WHERE account_monitor_id = ANY($1)
		) t
		WHERE rn <= $2
		ORDER BY account_monitor_id, checked_at DESC`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(monitorIDs), limit)
	if err != nil {
		return nil, fmt.Errorf("query recent account monitor checks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			c       service.AccountMonitorCheck
			latency sql.NullInt64
			ping    sql.NullInt64
		)
		if err := rows.Scan(&c.AccountMonitorID, &c.Model, &c.Status, &latency, &ping, &c.Message, &c.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan recent account monitor check: %w", err)
		}
		if latency.Valid {
			v := int(latency.Int64)
			c.LatencyMs = &v
		}
		if ping.Valid {
			v := int(ping.Int64)
			c.PingLatencyMs = &v
		}
		cp := c
		out[c.AccountMonitorID] = append(out[c.AccountMonitorID], &cp)
	}
	return out, rows.Err()
}

// Availability1h 计算每个 monitorID 近 1 小时可用率（operational+degraded 占比百分比）。
func (r *accountMonitorRepository) Availability1h(ctx context.Context, monitorIDs []int64) (map[int64]float64, error) {
	out := make(map[int64]float64, len(monitorIDs))
	if len(monitorIDs) == 0 {
		return out, nil
	}
	const q = `
		SELECT account_monitor_id,
		       COUNT(*)                                                       AS total,
		       COUNT(*) FILTER (WHERE status IN ('operational','degraded'))   AS ok
		FROM account_monitor_checks
		WHERE account_monitor_id = ANY($1)
		  AND checked_at >= NOW() - INTERVAL '1 hour'
		GROUP BY account_monitor_id`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(monitorIDs))
	if err != nil {
		return nil, fmt.Errorf("query account monitor availability: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			id    int64
			total int64
			ok    int64
		)
		if err := rows.Scan(&id, &total, &ok); err != nil {
			return nil, fmt.Errorf("scan account monitor availability: %w", err)
		}
		if total > 0 {
			out[id] = float64(ok) * 100.0 / float64(total)
		}
	}
	return out, rows.Err()
}

// AvgLatency1h 返回每个 monitorID 近 1 小时所有探测的平均 latency_ms。
// 仅统计 latency_ms 非空的样本；某 monitorID 无样本时不出现在结果里（调用方按缺失处理）。
func (r *accountMonitorRepository) AvgLatency1h(ctx context.Context, monitorIDs []int64) (map[int64]float64, error) {
	out := make(map[int64]float64, len(monitorIDs))
	if len(monitorIDs) == 0 {
		return out, nil
	}
	const q = `
		SELECT account_monitor_id, AVG(latency_ms)::float8 AS avg_latency
		FROM account_monitor_checks
		WHERE account_monitor_id = ANY($1)
		  AND checked_at >= NOW() - INTERVAL '1 hour'
		  AND latency_ms IS NOT NULL
		GROUP BY account_monitor_id`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(monitorIDs))
	if err != nil {
		return nil, fmt.Errorf("query account monitor avg latency: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			id  int64
			avg float64
		)
		if err := rows.Scan(&id, &avg); err != nil {
			return nil, fmt.Errorf("scan account monitor avg latency: %w", err)
		}
		out[id] = avg
	}
	return out, rows.Err()
}

func (r *accountMonitorRepository) DeleteChecksOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM account_monitor_checks WHERE checked_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old account monitor checks: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// ---------- mapping ----------

func entToServiceAccountMonitor(row *dbent.AccountMonitor) *service.AccountMonitor {
	return &service.AccountMonitor{
		ID:              row.ID,
		AccountID:       row.AccountID,
		Provider:        string(row.Provider),
		Model:           row.Model,
		Enabled:         row.Enabled,
		IntervalSeconds: row.IntervalSeconds,
		JitterSeconds:   row.JitterSeconds,
		LastCheckedAt:   row.LastCheckedAt,
		CreatedBy:       row.CreatedBy,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func entsToServiceAccountMonitors(rows []*dbent.AccountMonitor) []*service.AccountMonitor {
	out := make([]*service.AccountMonitor, 0, len(rows))
	for _, row := range rows {
		out = append(out, entToServiceAccountMonitor(row))
	}
	return out
}
