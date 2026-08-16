// 平台用户表。
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SysUser struct {
	ent.Schema
}

func (SysUser) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.String("username").Unique().Comment("用户名"),
		field.String("password").Comment("密码（bcrypt 哈希，禁止回传）"),
		field.String("nickname").Default("").Comment("昵称"),
		field.String("phone").Optional().Nillable().Comment("手机号"),
		field.String("email").Optional().Nillable().Comment("邮箱"),
		field.String("status").Default("0").Comment("状态（0正常 1停用）"),
		field.String("remark").Optional().Nillable().Comment("备注"),
	}
}

func (SysUser) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}

func (SysUser) Indexes() []ent.Index {
	return []ent.Index{
		// 列表查询路径（docs/03 §4）：状态 + 创建时间
		index.Fields("status", "created_at"),
	}
}
