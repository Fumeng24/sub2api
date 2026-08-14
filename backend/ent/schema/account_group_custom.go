package schema

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

func accountGroupFieldsCustom(upstream []ent.Field) []ent.Field {
	custom := []ent.Field{
		field.String("role").
			Default("primary").
			Validate(func(v string) error {
				switch v {
				case "primary", "backup":
					return nil
				default:
					return fmt.Errorf("invalid account group role %q", v)
				}
			}),
		field.Int("weight").
			Default(100).
			Positive(),
		field.Int("sort_order").
			Default(50),
		field.Bool("scheduling_configured").
			Default(false),
	}
	if len(upstream) == 0 {
		return custom
	}
	out := make([]ent.Field, 0, len(upstream)+len(custom))
	out = append(out, upstream[:len(upstream)-1]...)
	out = append(out, custom...)
	return append(out, upstream[len(upstream)-1])
}

func accountGroupIndexesCustom(upstream []ent.Index) []ent.Index {
	custom := index.Fields("group_id", "role", "sort_order")
	if len(upstream) == 0 {
		return []ent.Index{custom}
	}
	out := make([]ent.Index, 0, len(upstream)+1)
	out = append(out, upstream[:len(upstream)-1]...)
	out = append(out, custom)
	return append(out, upstream[len(upstream)-1])
}
