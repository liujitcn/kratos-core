package models

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

// TableNameBaseLog 是系统日志表名。
const TableNameBaseLog = "base_log"

// BaseLog 保存 Core 访问日志字段。
type BaseLog struct {
	ID             int64                 `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:系统日志iD" json:"id"`
	RequestID      string                `gorm:"column:request_id;type:varchar(64);comment:请求ID" json:"request_id"`
	RequestTime    time.Time             `gorm:"column:request_time;type:datetime;not null;index:idx_base_log_request_time,priority:1;comment:请求时间" json:"request_time"`
	Method         string                `gorm:"column:method;type:varchar(16);comment:请求方法" json:"method"`
	Operation      string                `gorm:"column:operation;type:varchar(128);not null;comment:操作方法" json:"operation"`
	Path           string                `gorm:"column:path;type:varchar(512);comment:请求路径" json:"path"`
	Referer        string                `gorm:"column:referer;type:varchar(1024);comment:请求源" json:"referer"`
	RequestURI     string                `gorm:"column:request_uri;type:varchar(2048);comment:请求URI" json:"request_uri"`
	RequestBody    string                `gorm:"column:request_body;type:text;comment:请求体" json:"request_body"`
	RequestHeader  string                `gorm:"column:request_header;type:text;comment:请求头" json:"request_header"`
	Response       string                `gorm:"column:response;type:text;comment:响应信息" json:"response"`
	CostTime       int64                 `gorm:"column:cost_time;type:bigint;not null;comment:操作耗时" json:"cost_time"`
	IsSuccess      bool                  `gorm:"column:is_success;type:tinyint(1);not null;comment:操作成功" json:"is_success"`
	StatusCode     int32                 `gorm:"column:status_code;type:int;not null;comment:状态码" json:"status_code"`
	Reason         string                `gorm:"column:reason;type:text;comment:操作失败原因" json:"reason"`
	Location       string                `gorm:"column:location;type:varchar(255);comment:操作地理位置" json:"location"`
	UserID         int64                 `gorm:"column:user_id;type:bigint;comment:操作者用户ID" json:"user_id"`
	UserName       string                `gorm:"column:user_name;type:varchar(64);comment:操作者账号名" json:"user_name"`
	ClientIP       string                `gorm:"column:client_ip;type:varchar(64);comment:操作者IP" json:"client_ip"`
	UserAgent      string                `gorm:"column:user_agent;type:varchar(1024);comment:浏览器的用户代理信息" json:"user_agent"`
	BrowserName    string                `gorm:"column:browser_name;type:varchar(64);comment:浏览器名称" json:"browser_name"`
	BrowserVersion string                `gorm:"column:browser_version;type:varchar(64);comment:浏览器版本" json:"browser_version"`
	ClientID       string                `gorm:"column:client_id;type:varchar(128);comment:客户端ID" json:"client_id"`
	ClientName     string                `gorm:"column:client_name;type:varchar(128);comment:客户端名称" json:"client_name"`
	OsName         string                `gorm:"column:os_name;type:varchar(64);comment:操作系统名称" json:"os_name"`
	OsVersion      string                `gorm:"column:os_version;type:varchar(64);comment:操作系统版本" json:"os_version"`
	DeletedAt      soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint unsigned;not null;comment:删除时间;softDelete:milli" json:"deleted_at"`
}

// TableName 返回系统日志表名。
func (*BaseLog) TableName() string { return TableNameBaseLog }

// TableComment 返回系统日志表注释。
func (*BaseLog) TableComment() string { return "系统日志信息" }
