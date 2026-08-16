// 角色-菜单关联表（docs/03 §6 sys_role_menus）
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SysRoleMenu struct {
	ent.Schema
}

func (SysRoleMenu) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.Int64("role_id").Comment("角色ID"),
		field.Int64("menu_id").Comment("菜单ID"),
	}
}

func (SysRoleMenu) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}

func (SysRoleMenu) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("role_id", "menu_id").Unique(),
		index.Fields("menu_id"),
	}
}
