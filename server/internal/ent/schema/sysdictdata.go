// 字典数据表（docs/03 §6 sys_dict_datas）
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SysDictData struct {
	ent.Schema
}

func (SysDictData) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.Int("sort").Default(0).Comment("字典排序"),
		field.String("label").Comment("字典标签"),
		field.String("value").Comment("字典键值"),
		field.String("dict_type").Comment("所属字典类型"),
		field.String("status").Default("0").Comment("状态（0正常 1停用）"),
		field.String("remark").Optional().Nillable().Comment("备注"),
	}
}

func (SysDictData) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}

func (SysDictData) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("dict_type", "sort"),
	}
}
