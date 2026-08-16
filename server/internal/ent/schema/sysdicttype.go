// 字典类型表（docs/03 §6 sys_dict_types）
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type SysDictType struct {
	ent.Schema
}

func (SysDictType) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.String("name").Comment("字典名称"),
		field.String("type").Unique().Comment("字典类型（如 sys_user_sex）"),
		field.String("status").Default("0").Comment("状态（0正常 1停用）"),
		field.String("remark").Optional().Nillable().Comment("备注"),
	}
}

func (SysDictType) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}
