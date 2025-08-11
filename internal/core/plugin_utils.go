package core

import (
	"context"
	"sync"
	"time"
)

// HostInfo 主机信息结构体（参考fscan）
type HostInfo struct {
	Host     string
	Port     int
	Timeout  time.Duration
	Retries  int
	Service  string
	Username string
	Password string
	Context  context.Context // 新增：支持上下文传递
}

// ScanResult 扫描结果
type ScanResult struct {
	Host      string            `json:"host"`
	Port      int               `json:"port"`
	Service   string            `json:"service"`
	Username  string            `json:"username"`
	Password  string            `json:"password"`
	Success   bool              `json:"success"`
	VulnType  string            `json:"vuln_type"` // "unauth", "weak_password", "vuln"
	Timestamp time.Time         `json:"timestamp"`
	Duration  time.Duration     `json:"duration"`
	Error     string            `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// PluginFunc 插件函数类型
type PluginFunc func(info *HostInfo) error

// PluginRegistry 插件注册表
type PluginRegistry struct {
	plugins map[string]PluginFunc
	mu      sync.RWMutex
}

// NewPluginRegistry 创建插件注册表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]PluginFunc),
	}
}

// Register 注册插件
func (r *PluginRegistry) Register(name string, plugin PluginFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[name] = plugin
}

// Get 获取插件
func (r *PluginRegistry) Get(name string) (PluginFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, exists := r.plugins[name]
	return plugin, exists
}

// List 列出所有插件
func (r *PluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// 全局插件注册表
var GlobalRegistry = NewPluginRegistry()
