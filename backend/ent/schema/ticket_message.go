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

// TicketMessage holds public replies and admin-only internal notes.
type TicketMessage struct {
	ent.Schema
}

func (TicketMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ticket_messages"},
	}
}

func (TicketMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("ticket_id"),
		field.String("sender_type").
			MaxLen(20).
			Default(domain.TicketMessageSenderUser).
			Comment("发送方: user, admin, system"),
		field.Int64("sender_id").
			Optional().
			Nillable().
			Comment("发送方用户ID"),
		field.String("sender_name").
			MaxLen(100).
			Default("").
			Comment("发送方名称快照"),
		field.String("visibility").
			MaxLen(20).
			Default(domain.TicketMessageVisibilityPublic).
			Comment("可见性: public, internal"),
		field.String("body").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty().
			Comment("消息正文（Markdown）"),
		field.JSON("attachments", []domain.TicketAttachment{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("附件链接列表"),
		field.Time("edited_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
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

func (TicketMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("messages").
			Field("ticket_id").
			Unique().
			Required(),
	}
}

func (TicketMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id"),
		index.Fields("created_at"),
		index.Fields("sender_type"),
		index.Fields("visibility"),
		index.Fields("ticket_id", "created_at"),
	}
}
