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

// TicketRead records the latest message an actor has read on a ticket.
type TicketRead struct {
	ent.Schema
}

func (TicketRead) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ticket_reads"},
	}
}

func (TicketRead) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("ticket_id"),
		field.String("actor_type").
			MaxLen(20).
			Default(domain.TicketReadActorUser).
			Comment("已读主体: user, admin"),
		field.Int64("actor_id").
			Comment("已读主体用户ID"),
		field.Int64("last_read_message_id").
			Optional().
			Nillable().
			Comment("最后已读消息ID"),
		field.Time("read_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("最后已读时间"),
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

func (TicketRead) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("reads").
			Field("ticket_id").
			Unique().
			Required(),
	}
}

func (TicketRead) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id"),
		index.Fields("actor_type", "actor_id"),
		index.Fields("ticket_id", "actor_type", "actor_id").Unique(),
	}
}
