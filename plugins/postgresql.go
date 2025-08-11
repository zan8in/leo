package plugins

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/zan8in/leo/internal/core"
)

// PostgresqlScan PostgreSQL数据库扫描函数
func PostgresqlScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 5432 // PostgreSQL默认端口
	}

	// 获取context，如果没有则创建默认的
	ctx := info.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	// 使用超时时间，如果为0则使用默认值
	timeout := info.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// 创建带超时的context用于单个请求
	requestCtx, requestCancel := context.WithTimeout(ctx, timeout)
	defer requestCancel()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 尝试PostgreSQL认证
	return postgresqlAuth(requestCtx, info)
}

// postgresqlAuth PostgreSQL认证函数
func postgresqlAuth(ctx context.Context, info *core.HostInfo) error {
	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 设置连接超时
	timeout := info.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// 尝试连接不同的数据库
	databases := []string{"postgres", "template1", "template0"}

	for _, dbname := range databases {
		// 检查context是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := tryPostgresqlConnect(ctx, info, dbname, timeout); err == nil {
			fmt.Printf("[+] %s:%d postgresql %s:%s\n", info.Host, info.Port, info.Username, info.Password)
			return nil
		}
	}

	return fmt.Errorf("postgresql auth failed: all connection attempts failed")
}

// tryPostgresqlConnect 尝试连接PostgreSQL
func tryPostgresqlConnect(ctx context.Context, info *core.HostInfo, dbname string, timeout time.Duration) error {
	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// PostgreSQL连接字符串格式
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=%d",
		info.Host, info.Port, info.Username, info.Password, dbname, int(timeout.Seconds()))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// 使用传入的context进行连接测试
	return db.PingContext(ctx)
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("postgresql", PostgresqlScan)
}
