package plugins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zan8in/leo/internal/core"
)

// MysqlScan MySQL扫描函数（参考fscan设计）
func MysqlScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 3306 // MySQL默认端口
	}

	if info.Username == "" && info.Password == "" {
		return errors.New("mysql username and password are empty")
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

	fmt.Println(info.Username, info.Password)

	// MySQL连接字符串格式
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8&timeout=%s&readTimeout=%s&writeTimeout=%s",
		info.Username, info.Password, info.Host, info.Port, timeout, timeout, timeout)

	db, err := sql.Open("mysql", dsn)
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
		fmt.Printf("[+] %s:%d mysql %s:%s\n", info.Host, info.Port, info.Username, info.Password)
	}

	return err
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("mysql", MysqlScan)
}
