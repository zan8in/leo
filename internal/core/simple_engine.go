package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zan8in/leo/internal/plugin"
)

// EngineConfig 引擎配置
type EngineConfig struct {
	Concurrency int           `json:"concurrency"` // 并发数
	Timeout     time.Duration `json:"timeout"`     // 超时时间
	Retries     int           `json:"retries"`     // 重试次数
	Verbose     bool          `json:"verbose"`     // 详细输出
	FullScan    bool          `json:"fullscan"`    // 全扫描模式
}

// Task 认证任务
type Task struct {
	Service  string        `json:"service"`  // 服务类型
	Target   plugin.Target `json:"target"`   // 目标信息
	Username string        `json:"username"` // 用户名
	Password string        `json:"password"` // 密码
}

// SimpleEngine 简化版引擎，不使用连接池
type SimpleEngine struct {
	pluginMgr    *plugin.Manager
	config       EngineConfig
	foundTargets sync.Map // 记录已找到弱口令的目标
}

func NewSimpleEngine(pluginMgr *plugin.Manager, config EngineConfig) *SimpleEngine {
	return &SimpleEngine{
		pluginMgr: pluginMgr,
		config:    config,
	}
}

// RunByTargets 按目标分组运行任务
func (e *SimpleEngine) RunByTargets(taskGroups map[string][]Task) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, e.config.Concurrency)

	// 添加延迟控制，避免瞬间大量连接
	delay := time.Duration(0)
	if e.config.Concurrency > 50 {
		// 当并发数超过50时，添加延迟
		delay = time.Millisecond * time.Duration(2000/e.config.Concurrency)
		if delay < time.Millisecond {
			delay = time.Millisecond
		}
	}

	taskIndex := 0
	for targetKey, tasks := range taskGroups {
		for _, task := range tasks {
			// 检查该目标是否已找到弱口令
			if !e.config.FullScan {
				if _, found := e.foundTargets.Load(targetKey); found {
					continue // 跳过该目标的剩余任务
				}
			}

			wg.Add(1)

			// 添加启动延迟
			if delay > 0 && taskIndex > 0 {
				time.Sleep(delay)
			}
			taskIndex++

			go func(t Task, tKey string) {
				defer func() {
					wg.Done()
					// 捕获panic，避免程序崩溃
					if r := recover(); r != nil {
						if e.config.Verbose {
							fmt.Printf("[PANIC] %s://%s:%d %s:%s - %v\n",
								t.Service, t.Target.Host, t.Target.Port,
								t.Username, t.Password, r)
						}
					}
				}()

				semaphore <- struct{}{}        // 获取信号量
				defer func() { <-semaphore }() // 释放信号量

				e.executeSimpleTask(t, tKey)
			}(task, targetKey)
		}
	}

	wg.Wait()
	return nil
}

// Run 保持原有接口兼容性
func (e *SimpleEngine) Run(tasks []Task) error {
	// 将任务按目标分组
	taskGroups := make(map[string][]Task)
	for _, task := range tasks {
		targetKey := fmt.Sprintf("%s:%d", task.Target.Host, task.Target.Port)
		taskGroups[targetKey] = append(taskGroups[targetKey], task)
	}
	return e.RunByTargets(taskGroups)
}

func (e *SimpleEngine) executeSimpleTask(task Task, targetKey string) {
	// 再次检查该目标是否已找到弱口令
	if !e.config.FullScan {
		if _, found := e.foundTargets.Load(targetKey); found {
			return // 该目标已找到弱口令，跳过
		}
	}

	start := time.Now()
	result := plugin.AuthResult{
		Target:    task.Target,
		Username:  task.Username,
		Password:  task.Password,
		Service:   task.Service,
		Timestamp: start,
		Success:   false,
	}

	defer func() {
		result.Duration = time.Since(start)
		if result.Success {
			// 在非全扫描模式下，检查是否已经输出过该目标的成功信息
			if !e.config.FullScan {
				if _, loaded := e.foundTargets.LoadOrStore(targetKey, true); loaded {
					// 已经输出过了，跳过
					return
				}
			}

			// SUCCESS 信息始终显示
			fmt.Printf("[SUCCESS] %s://%s:%d %s:%s\n",
				result.Service, result.Target.Host, result.Target.Port,
				result.Username, result.Password)
		} else if e.config.Verbose {
			// FAILED 信息只在 verbose 模式下显示
			fmt.Printf("[FAILED] %s://%s:%d %s:%s - %s\n",
				result.Service, result.Target.Host, result.Target.Port,
				result.Username, result.Password, result.Error)
		}
	}()

	// 获取插件
	plugin, err := e.pluginMgr.Get(task.Service)
	if err != nil {
		result.Error = fmt.Sprintf("plugin not found: %v", err)
		return
	}

	// 实现重试机制，增加错误处理
	var lastErr error
	for attempt := 0; attempt <= e.config.Retries; attempt++ {
		if attempt > 0 {
			// 重试前等待一段时间，递增延迟
			time.Sleep(time.Millisecond * time.Duration(200*attempt))
		}

		// 使用更短的超时时间避免长时间等待
		timeout := e.config.Timeout
		if timeout > 10*time.Second {
			timeout = 10 * time.Second
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		// 添加错误处理
		func() {
			defer func() {
				cancel()
				if r := recover(); r != nil {
					lastErr = fmt.Errorf("panic during connection: %v", r)
				}
			}()

			conn, err := plugin.Connect(ctx, task.Target)
			if err != nil {
				lastErr = fmt.Errorf("failed to connect: %v", err)
				return
			}
			defer conn.Close()

			err = conn.Auth(task.Username, task.Password)
			if err == nil {
				result.Success = true
				return
			}

			lastErr = err
		}()

		if result.Success {
			return
		}
	}

	if lastErr != nil {
		result.Error = lastErr.Error()
	}
}
