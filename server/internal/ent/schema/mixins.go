// 公共字段 Mixin（docs/03 §3）：审计时间、软删除、操作人、备注
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

type BaseMixin struct {
	mixin.Schema
}

func (BaseMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("created_by").Default("").Comment("创建者"),
		field.String("updated_by").Default("").Comment("更新者"),
		// 注意：Immutable 时间字段不会被 ent 自动填充，必须显式 Default/UpdateDefault
		field.Time("created_at").Immutable().Default(time.Now).Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
		field.Time("deleted_at").Optional().Nillable().Comment("软删除时间"),
	}
}
