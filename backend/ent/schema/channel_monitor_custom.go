package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

func channelMonitorFieldsCustom(upstream []ent.Field) []ent.Field {
	apiKeyID := field.Int64("api_key_id").
		Optional().
		Nillable().
		Comment("Linked user API key ID; runtime prefers the current api_keys.key when present")
	sortOrder := field.Int("sort_order").
		Default(0).
		Comment("Display order for channel monitor lists; lower values appear first")
	out := make([]ent.Field, 0, len(upstream)+2)
	for _, item := range upstream {
		out = append(out, item)
		switch item.Descriptor().Name {
		case "api_key_encrypted":
			out = append(out, apiKeyID)
		case "group_name":
			out = append(out, sortOrder)
		}
	}
	return out
}

func channelMonitorEdgesCustom(upstream []ent.Edge) []ent.Edge {
	apiKey := edge.To("api_key", APIKey.Type).
		Field("api_key_id").
		Unique().
		Annotations(entsql.OnDelete(entsql.SetNull))
	out := make([]ent.Edge, 0, len(upstream)+1)
	for _, item := range upstream {
		out = append(out, item)
		if item.Descriptor().Name == "daily_rollups" {
			out = append(out, apiKey)
		}
	}
	return out
}

func channelMonitorIndexesCustom(upstream []ent.Index) []ent.Index {
	out := make([]ent.Index, 0, len(upstream)+2)
	for _, item := range upstream {
		out = append(out, item)
		fields := item.Descriptor().Fields
		if len(fields) == 1 && fields[0] == "group_name" {
			out = append(out, index.Fields("sort_order", "id"))
		}
		if len(fields) == 1 && fields[0] == "template_id" {
			out = append(out, index.Fields("api_key_id"))
		}
	}
	return out
}
