// 菜单表（docs/03 §6 sys_menus；前端动态路由来源，docs/05 §5）
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SysMenu struct {
	ent.Schema
}

func (SysMenu) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.Int64("parent_id").Default(0).Comment("父菜单ID（0为根）"),
		field.String("name").Comment("菜单名称"),
		field.String("menu_type").Default("C").Comment("菜单类型（M目录 C菜单 F按钮）"),
		field.String("path").Default("").Comment("路由地址"),
		field.String("component").Optional().Nillable().Comment("前端组件路径（如 system/user/index）"),
		field.String("perms").Optional().Nillable().Comment("权限字符串（如 system:user:list）"),
		field.String("icon").Optional().Nillable().Comment("图标名（lucide）"),
		field.Int("order_num").Default(0).Comment("显示顺序"),
		field.Bool("visible").Default(true).Comment("是否显示（false 时不出现在侧栏）"),
		field.String("status").Default("0").Comment("状态（0正常 1停用）"),
		field.String("remark").Optional().Nillable().Comment("备注"),
	}
}

func (SysMenu) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}

func (SysMenu) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("parent_id", "order_num"),
		index.Fields("perms"),
	}
}
