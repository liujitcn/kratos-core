package models

import "time"

// TableNameBaseJobLog 是定时任务日志表名。
const TableNameBaseJobLog = "base_job_log"

// BaseJobLog 保存定时任务执行日志。
type BaseJobLog struct {
	ID          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:任务日志ID" json:"id"`
	JobID       int64     `gorm:"column:job_id;type:bigint;not null;index:idx_base_job_log_job_id,priority:1;comment:任务ID" json:"job_id"`
	Input       string    `gorm:"column:input;type:varchar(1024);comment:执行参数" json:"input"`
	Output      string    `gorm:"column:output;type:text;comment:输出结果" json:"output"`
	Error       string    `gorm:"column:error;type:text;comment:错误信息" json:"error"`
	Status      int32     `gorm:"column:status;type:tinyint;not null;comment:状态：枚举【BaseJobLogStatus】" json:"status"`
	ProcessTime int32     `gorm:"column:process_time;type:int;not null;comment:消耗时间/毫秒" json:"process_time"`
	ExecuteTime time.Time `gorm:"column:execute_time;type:datetime;not null;comment:执行时间" json:"execute_time"`
}

// TableName 返回定时任务日志表名。
func (*BaseJobLog) TableName() string { return TableNameBaseJobLog }

// TableComment 返回定时任务日志表注释。
func (*BaseJobLog) TableComment() string { return "定时任务日志信息" }
