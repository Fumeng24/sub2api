package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AccountMonitor holds the schema definition for the AccountMonitor entity.
// 账号监控：为单个 api_key 类上游账号配置的探针。与 channel-monitor 完全独立，
// 纯管理员视角，永不暴露给用户端。运行时按 account_id 实时读取该账号的
// credentials.api_key + credentials.base_url 作为探测凭证（不落库明文/密文）。
type AccountMonitor struct {
	ent.Schema
}

func (AccountMonitor) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_monitors"},
	}
}

func (AccountMonitor) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (AccountMonitor) Fields() []ent.Field {
	return []ent.Field{
		// account_id: 被监控的账号（仅 api_key 类）。必填——这套监控离开账号无意义。
		field.Int64("account_id"),
		field.Enum("provider").
			Values("openai", "anthropic", "gemini").
			Default("openai"),
		// model: 探测用模型，默认 gpt-5.4-mini，admin 可改。
		field.String("model").
			NotEmpty().
			Default("gpt-5.4-mini").
			MaxLen(200),
		field.Bool("enabled").
			Default(true),
		field.Int("interval_seconds").
			Default(60).
			Range(15, 3600),
		field.Int("jitter_seconds").
			Default(0).
			Range(0, 3600).
			Comment("每次调度在 interval 基础上 ± [0, jitter] 的均匀随机偏移（秒）；0 表示固定间隔"),
		field.Time("last_checked_at").
			Optional().
			Nillable(),
		field.Int64("created_by"),
	}
}

func (AccountMonitor) Edges() []ent.Edge {
	return []ent.Edge{
		// 探测历史：监控删除时级联清除。
		edge.To("checks", AccountMonitorCheck.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		// 绑定账号：账号被删除时本监控连带删除（监控离开账号无意义）。
		edge.To("account", Account.Type).
			Field("account_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (AccountMonitor) Indexes() []ent.Index {
	return []ent.Index{
		// 一个账号至多一个监控。
		index.Fields("account_id").Unique(),
		// 调度器扫描到期监控。
		index.Fields("enabled", "last_checked_at"),
	}
}
