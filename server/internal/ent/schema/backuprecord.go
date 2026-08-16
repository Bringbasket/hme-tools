// 数据备份记录表：备份文件名、状态、触发方式和过期时间。
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type BackupRecord struct {
	ent.Schema
}

func (BackupRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable().Comment("ID"),
		field.String("record_key").Unique().Comment("备份记录短ID（8位hex）"),
		field.String("status").Default("running").Comment("状态 running/completed/failed"),
		field.String("file_name").Default("").Comment("备份文件名（S3 key）"),
		field.Int64("size_bytes").Default(0).Comment("文件大小（字节）"),
		field.Int("parts").Default(1).Comment("分卷数"),
		field.Time("expire_at").Optional().Nillable().Comment("过期时间（过期天数换算，0天=空）"),
		field.String("trigger_type").Default("manual").Comment("触发方式 manual 手动 / scheduled 定时"),
		field.Time("started_at").Comment("开始时间"),
		field.Time("finished_at").Optional().Nillable().Comment("完成时间"),
		field.String("error_message").Optional().Nillable().Comment("失败原因"),
		field.Float("duration_ms").
			SchemaType(map[string]string{dialect.Postgres: "numeric(12,2)"}).
			Default(0).
			Comment("耗时(毫秒)"),
	}
}

func (BackupRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{BaseMixin{}}
}

func (BackupRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "created_at"),
		index.Fields("expire_at"),
	}
}
