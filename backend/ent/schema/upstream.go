package schema

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Upstream stores one upstream site's management identity. Runtime accounts
// remain platform-specific and reference this record through upstream_id.
type Upstream struct {
	ent.Schema
}

func (Upstream) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "upstreams"}}
}

func (Upstream) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}, mixins.SoftDeleteMixin{}}
}

func (Upstream) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(100).NotEmpty().
			Validate(func(value string) error {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("upstream name cannot be empty")
				}
				return nil
			}),
		field.String("base_url").NotEmpty().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Enum("kind").Values("auto", "newapi", "sub2api").Default("auto"),
		field.JSON("credentials", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int64("proxy_id").Optional().Nillable(),
		field.Enum("status").Values("unknown", "healthy", "degraded", "error").Default("unknown"),
		field.Time("last_probe_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("last_probe_error").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.JSON("metadata", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
	}
}

func (Upstream) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("proxy", Proxy.Type).Field("proxy_id").Unique(),
		edge.To("accounts", Account.Type),
	}
}

func (Upstream) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
		index.Fields("kind"),
		index.Fields("base_url"),
		index.Fields("proxy_id"),
		index.Fields("status"),
		index.Fields("deleted_at"),
	}
}
