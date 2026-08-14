package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AccountMonitorCheck holds the schema definition for the AccountMonitorCheck entity.
// 账号监控探测历史：每次探测一行。保留趋势供 admin 视图查可用率；超期由清理任务批量删。
type AccountMonitorCheck struct {
	ent.Schema
}

func (AccountMonitorCheck) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_monitor_checks"},
	}
}

func (AccountMonitorCheck) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_monitor_id"),
		field.String("model").
			NotEmpty().
			MaxLen(200),
		field.Enum("status").
			Values("operational", "degraded", "failed", "error"),
		field.Int("latency_ms").
			Optional().
			Nillable(),
		field.Int("ping_latency_ms").
			Optional().
			Nillable(),
		field.String("message").
			Optional().
			Default("").
			MaxLen(500),
		field.Time("checked_at").
			Default(time.Now),
	}
}

func (AccountMonitorCheck) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("monitor", AccountMonitor.Type).
			Ref("checks").
			Field("account_monitor_id").
			Unique().
			Required(),
	}
}

func (AccountMonitorCheck) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_monitor_id", "checked_at"),
		index.Fields("checked_at"),
	}
}
