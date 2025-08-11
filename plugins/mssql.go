package plugins

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/zan8in/leo/internal/core"
)

// MssqlScan MSSQL扫描函数（参考fscan设计）
func MssqlScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 1433 // MSSQL默认端口
	}

	// 获取context，如果没有则创建默认的
	ctx := info.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
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

	// MSSQL连接字符串格式
	dsn := fmt.Sprintf("server=%s;port=%d;user id=%s;password=%s;database=master;connection timeout=%d",
		info.Host, info.Port, info.Username, info.Password, int(timeout.Seconds()))

	db, err := sql.Open("mssql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 使用请求级context进行连接测试
	err = db.PingContext(requestCtx)
	if err == nil {
		// 认证成功，输出结果
		fmt.Printf("[+] %s:%d mssql %s:%s\n", info.Host, info.Port, info.Username, info.Password)
	}

	return err
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("mssql", MssqlScan)
}
