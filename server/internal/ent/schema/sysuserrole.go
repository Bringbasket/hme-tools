// 用户-角色关联表（docs/03 §6 sys_user_roles）
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SysUserRole struct {
	ent.Schema
}

func (SysUserRole) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.Int64("user_id").Comment("用户ID"),
		field.Int64("role_id").Comment("角色ID"),
	}
}

func (SysUserRole) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}

func (SysUserRole) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "role_id").Unique(),
		index.Fields("role_id"),
	}
}
