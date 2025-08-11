package plugin

import (
	"fmt"
	"sync"
)

// Manager 插件管理器
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	configs map[string]map[string]interface{}
}

// NewManager 创建插件管理器
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		configs: make(map[string]map[string]interface{}),
	}
}

// Register 注册插件
func (m *Manager) Register(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	name := plugin.Name()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	
	m.plugins[name] = plugin
	return nil
}

// Get 获取插件
func (m *Manager) Get(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	
	return plugin, nil
}

// List 列出所有插件
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}

// LoadConfig 加载插件配置
func (m *Manager) LoadConfig(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.configs[name] = config
	
	if plugin, exists := m.plugins[name]; exists {
		return plugin.Init(config)
	}
	
	return nil
}

// InitAll 初始化所有插件
func (m *Manager) InitAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for name, plugin := range m.plugins {
		config := m.configs[name]
		if config == nil {
			config = make(map[string]interface{})
		}
		
		if err := plugin.Init(config); err != nil {
			return fmt.Errorf("failed to init plugin %s: %w", name, err)
		}
	}
	
	return nil
}

// Close 关闭所有插件
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for name, plugin := range m.plugins {
		if err := plugin.Close(); err != nil {
			fmt.Printf("Warning: failed to close plugin %s: %v\n", name, err)
		}
	}
	
	return nil
}