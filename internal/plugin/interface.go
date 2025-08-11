package plugin

import (
	"context"
	"time"
)

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string
	
	// Version 返回插件版本
	Version() string
	
	// Init 初始化插件
	Init(config map[string]interface{}) error
	
	// Connect 创建连接
	Connect(ctx context.Context, target Target) (Connection, error)
	
	// Close 关闭插件
	Close() error
}

// Connection 连接接口
type Connection interface {
	// Auth 认证
	Auth(username, password string) error
	
	// Ping 测试连接
	Ping() error
	
	// Close 关闭连接
	Close() error
	
	// Info 获取连接信息
	Info() ConnectionInfo
}

// Target 目标配置
type Target struct {
	Host    string        `json:"host"`
	Port    int           `json:"port"`
	Timeout time.Duration `json:"timeout"`
	Retries int           `json:"retries"`
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	Service   string            `json:"service"`
	Host      string            `json:"host"`
	Port      int               `json:"port"`
	Connected bool              `json:"connected"`
	Metadata  map[string]string `json:"metadata"`
}

// AuthResult 认证结果
type AuthResult struct {
	Success   bool              `json:"success"`
	Target    Target            `json:"target"`
	Username  string            `json:"username"`
	Password  string            `json:"password"`
	Service   string            `json:"service"`
	Timestamp time.Time         `json:"timestamp"`
	Duration  time.Duration     `json:"duration"`
	Error     string            `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}