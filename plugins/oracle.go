package plugins

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
	"github.com/zan8in/leo/internal/core"
)

// OracleScan Oracle数据库扫描函数
func OracleScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 1521 // Oracle默认端口
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

	// 尝试Oracle认证
	return oracleAuth(requestCtx, info)
}

// oracleAuth Oracle认证函数
func oracleAuth(ctx context.Context, info *core.HostInfo) error {
	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 尝试多种连接方式，参考fscan的实现
	serviceNames := []string{"XE", "ORCL", "xe", "orcl", "XEPDB1", "ORCLPDB1"}

	// 首先尝试使用SERVICE_NAME连接
	for _, serviceName := range serviceNames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := tryOracleConnect(ctx, info, serviceName, false); err == nil {
			fmt.Printf("[+] %s:%d oracle %s:%s\n", info.Host, info.Port, info.Username, info.Password)
			return nil
		}
	}

	// 尝试使用SID连接
	for _, sid := range []string{"XE", "ORCL", "xe", "orcl"} {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := tryOracleConnectWithSID(ctx, info, sid); err == nil {
			fmt.Printf("[+] %s:%d oracle %s:%s\n", info.Host, info.Port, info.Username, info.Password)
			return nil
		}
	}

	// 尝试作为SYSDBA连接（如果用户名是sys）
	if info.Username == "sys" || info.Username == "SYS" {
		for _, serviceName := range serviceNames {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err := tryOracleConnect(ctx, info, serviceName, true); err == nil {
				fmt.Printf("[+] %s:%d oracle %s:%s (SYSDBA)\n", info.Host, info.Port, info.Username, info.Password)
				return nil
			}
		}
	}

	return fmt.Errorf("oracle auth failed: all connection attempts failed")
}

// tryOracleConnect 尝试使用SERVICE_NAME连接
func tryOracleConnect(ctx context.Context, info *core.HostInfo, serviceName string, asSysdba bool) error {
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

	urlOptions := map[string]string{
		"CONNECTION TIMEOUT": fmt.Sprintf("%.0f", timeout.Seconds()),
	}

	if asSysdba {
		urlOptions["SYSDBA"] = "true"
	}

	connStr := go_ora.BuildUrl(info.Host, info.Port, serviceName, info.Username, info.Password, urlOptions)

	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	// 使用传入的context进行连接测试
	return db.PingContext(ctx)
}

// tryOracleConnectWithSID 尝试使用SID连接
func tryOracleConnectWithSID(ctx context.Context, info *core.HostInfo, sid string) error {
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

	urlOptions := map[string]string{
		"SID":                sid,
		"CONNECTION TIMEOUT": fmt.Sprintf("%.0f", timeout.Seconds()),
	}

	connStr := go_ora.BuildUrl(info.Host, info.Port, "", info.Username, info.Password, urlOptions)

	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	// 使用传入的context进行连接测试
	return db.PingContext(ctx)
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("oracle", OracleScan)
}
