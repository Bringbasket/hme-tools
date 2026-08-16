// 操作日志表（docs/03 §6 monitor_oper_logs；日志类表不做软删除）
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SysOperLog struct {
	ent.Schema
}

func (SysOperLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.String("title").Default("").Comment("模块标题"),
		field.String("business_type").Default("OTHER").Comment("业务类型（OTHER/INSERT/UPDATE/DELETE/GRANT/EXPORT）"),
		field.String("method").Default("").Comment("请求方法"),
		field.String("path").Default("").Comment("请求路径"),
		field.Int64("operator_id").Optional().Nillable().Comment("操作人ID"),
		field.String("operator_name").Default("").Comment("操作人"),
		field.String("ip").Default("").Comment("操作IP"),
		field.Int("status_code").Default(0).Comment("响应状态码"),
		field.Int64("duration_ms").Default(0).Comment("耗时(毫秒)"),
		field.String("params").Optional().Nillable().Comment("请求参数摘要（敏感字段脱敏）"),
		field.String("error").Optional().Nillable().Comment("错误信息"),
		field.Time("created_at").Immutable().Default(time.Now).Comment("操作时间"),
	}
}

func (SysOperLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
		index.Fields("operator_id", "created_at"),
	}
}
