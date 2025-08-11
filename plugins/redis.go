package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/zan8in/leo/internal/core"
)

// RedisScan Redis扫描函数（参考fscan设计）
func RedisScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 6379 // Redis默认端口
	}

	// 获取context，如果没有则创建默认的
	ctx := info.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 优先检测未授权访问（无密码访问）
	if info.Username == "" && info.Password == "" {
		if err := redisUnauth(info, ctx); err == nil {
			fmt.Printf("[+] %s:%d redis unauthorized access\n", info.Host, info.Port)
			return nil // 发现未授权访问，停止进一步检测
		}
	}

	// 进行认证检测
	return redisAuth(info, ctx)
}

// redisUnauth 检测未授权访问（无密码）
func redisUnauth(info *core.HostInfo, parentCtx context.Context) error {
	timeout := info.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	// 创建带超时的context用于单个请求
	requestCtx, requestCancel := context.WithTimeout(parentCtx, timeout)
	defer requestCancel()

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)

	// 创建Redis客户端（无密码）
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "", // 无密码
		DB:           0,  // 默认数据库
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	})
	defer rdb.Close()

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	// 尝试ping测试连接
	_, err := rdb.Ping(requestCtx).Result()
	if err != nil {
		return err
	}

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	// 尝试执行一个简单的命令来验证访问权限
	_, err = rdb.Info(requestCtx).Result()
	return err
}

// redisAuth 认证检测
func redisAuth(info *core.HostInfo, parentCtx context.Context) error {
	timeout := info.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// 创建带超时的context用于单个请求
	requestCtx, requestCancel := context.WithTimeout(parentCtx, timeout)
	defer requestCancel()

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)

	// 创建Redis客户端（带密码）
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     info.Password, // Redis密码
		DB:           0,             // 默认数据库
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	})
	defer rdb.Close()

	// 检查context是否已取消
	select {
	case <-parentCtx.Done():
		return parentCtx.Err()
	default:
	}

	// 尝试ping测试连接
	_, err := rdb.Ping(requestCtx).Result()
	if err == nil {
		// 认证成功，输出结果
		if info.Password != "" {
			fmt.Printf("[+] %s:%d redis :%s\n", info.Host, info.Port, info.Password)
		} else {
			fmt.Printf("[+] %s:%d redis no password\n", info.Host, info.Port)
		}
	}

	return err
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("redis", RedisScan)
}
