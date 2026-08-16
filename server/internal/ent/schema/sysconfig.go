// 参数配置表（docs/03 §6 sys_configs）
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type SysConfig struct {
	ent.Schema
}

func (SysConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.String("name").Comment("参数名称"),
		field.String("key").Unique().Comment("参数键名（如 sys.account.registerUser）"),
		field.String("value").Comment("参数键值"),
		field.String("remark").Optional().Nillable().Comment("备注"),
	}
}

func (SysConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}
