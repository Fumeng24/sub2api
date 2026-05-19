package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Ticket holds the schema definition for support tickets.
//
// 删除策略：硬删除（消息和已读状态通过外键级联删除）。
type Ticket struct {
	ent.Schema
}

func (Ticket) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "tickets"},
	}
}

func (Ticket) Fields() []ent.Field {
	return []ent.Field{
		field.String("ticket_no").
			MaxLen(32).
			NotEmpty().
			Unique().
			Comment("工单编号"),
		field.Int64("user_id").
			Comment("提交用户ID"),
		field.String("user_email").
			MaxLen(255).
			Default("").
			Comment("提交用户邮箱快照"),
		field.String("user_name").
			MaxLen(100).
			Default("").
			Comment("提交用户名称快照"),
		field.String("subject").
			MaxLen(200).
			NotEmpty().
			Comment("工单标题"),
		field.String("category").
			MaxLen(50).
			Default(domain.TicketCategoryGeneral).
			Comment("分类: general, billing, usage, technical, account"),
		field.String("priority").
			MaxLen(20).
			Default(domain.TicketPriorityNormal).
			Comment("优先级: low, normal, high, urgent"),
		field.String("status").
			MaxLen(20).
			Default(domain.TicketStatusOpen).
			Comment("状态: open, pending, resolved, closed"),
		field.String("source").
			MaxLen(30).
			Default("user").
			Comment("来源"),
		field.String("template_key").
			MaxLen(80).
			Default("").
			Comment("工单模板标识"),
		field.String("context_type").
			MaxLen(50).
			Default("").
			Comment("关联上下文类型"),
		field.String("context_id").
			MaxLen(128).
			Default("").
			Comment("关联上下文ID"),
		field.JSON("context_data", map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("动态模板字段和上下文快照"),
		field.Int64("assignee_id").
			Optional().
			Nillable().
			Comment("指派管理员ID"),
		field.Time("escalated_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("升级给超级管理员时间"),
		field.Int64("escalated_by").
			Optional().
			Nillable().
			Comment("升级操作人ID"),
		field.String("escalation_reason").
			MaxLen(500).
			Default("").
			Comment("升级原因"),
		field.Time("sla_due_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("SLA响应截止时间"),
		field.Time("sla_reminded_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("SLA催办通知时间"),
		field.Time("last_message_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("最后消息时间"),
		field.Time("last_user_message_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("最后用户回复时间"),
		field.Time("last_admin_message_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("最后管理员回复时间"),
		field.Time("resolved_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("解决时间"),
		field.Time("closed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("关闭时间"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (Ticket) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("messages", TicketMessage.Type),
		edge.To("reads", TicketRead.Type),
	}
}

func (Ticket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("priority"),
		index.Fields("assignee_id"),
		index.Fields("template_key"),
		index.Fields("escalated_at"),
		index.Fields("sla_due_at"),
		index.Fields("sla_reminded_at"),
		index.Fields("last_message_at"),
		index.Fields("created_at"),
		index.Fields("status", "last_message_at"),
		index.Fields("user_id", "status"),
	}
}
