package models

import "gorm.io/plugin/soft_delete"

// TableNameBaseJob 是定时任务表名。
const TableNameBaseJob = "base_job"

// BaseJob 保存 Core 调度所需的定时任务字段。
type BaseJob struct {
	ID             int64                 `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:任务ID" json:"id"`
	InvokeTarget   string                `gorm:"column:invoke_target;type:varchar(100);not null;uniqueIndex:unique_base_job,priority:1;comment:调用目标" json:"invoke_target"`
	Args           string                `gorm:"column:args;type:json;comment:目标参数" json:"args"`
	CronExpression string                `gorm:"column:cron_expression;type:varchar(50);not null;comment:cron表达式" json:"cron_expression"`
	Status         int32                 `gorm:"column:status;type:tinyint;not null;comment:状态：枚举【Status】" json:"status"`
	EntryID        int32                 `gorm:"column:entry_id;type:smallint;comment:job启动时返回的id" json:"entry_id"`
	DeletedAt      soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint unsigned;not null;uniqueIndex:unique_base_job,priority:2;comment:删除时间;softDelete:milli" json:"deleted_at"`
}

// TableName 返回定时任务表名。
func (*BaseJob) TableName() string { return TableNameBaseJob }

// TableComment 返回定时任务表注释。
func (*BaseJob) TableComment() string { return "定时任务信息" }
