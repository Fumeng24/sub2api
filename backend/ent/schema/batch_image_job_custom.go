package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

func batchImageJobFieldsCustom(upstream []ent.Field) []ent.Field {
	parentBatchID := field.String("parent_batch_id").Optional().Nillable().MaxLen(64)
	pricingSnapshot := []ent.Field{
		field.Float("base_unit_price").SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).Default(0),
		field.Float("group_rate_multiplier").SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).Default(1.0),
		field.Float("account_rate_multiplier").SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).Default(1.0),
		field.Float("batch_discount_multiplier").SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).Default(0.5),
		field.Float("hold_multiplier").SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).Default(0.6),
		field.Float("billable_unit_price").SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).Default(0),
		field.Float("hold_unit_price").SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).Default(0),
		field.Int("pricing_snapshot_version").Default(0),
	}
	out := make([]ent.Field, 0, len(upstream)+1+len(pricingSnapshot))
	for _, item := range upstream {
		out = append(out, item)
		switch item.Descriptor().Name {
		case "task_name":
			out = append(out, parentBatchID)
		case "actual_cost":
			out = append(out, pricingSnapshot...)
		}
	}
	return out
}

func batchImageJobIndexesCustom(upstream []ent.Index) []ent.Index {
	out := make([]ent.Index, 0, len(upstream)+2)
	for _, item := range upstream {
		out = append(out, item)
		fields := item.Descriptor().Fields
		if len(fields) == 1 && fields[0] == "output_expires_at" {
			out = append(out,
				index.Fields("task_name"),
				index.Fields("parent_batch_id").Annotations(entsql.IndexWhere("parent_batch_id IS NOT NULL AND parent_batch_id <> ''")),
			)
		}
	}
	return out
}
