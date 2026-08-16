// 角色表（docs/03 §6 sys_roles）
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type SysRole struct {
	ent.Schema
}

func (SysRole) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.String("name").Unique().Comment("角色名称"),
		field.String("code").Unique().Comment("角色权限字符串（如 admin/common）"),
		field.Int("sort").Default(0).Comment("显示顺序"),
		field.Bool("is_admin").Default(false).Comment("超级管理员（跳过权限校验）"),
		field.String("status").Default("0").Comment("状态（0正常 1停用）"),
		field.String("remark").Optional().Nillable().Comment("备注"),
	}
}

func (SysRole) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}
