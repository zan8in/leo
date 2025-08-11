package plugins

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/zan8in/leo/internal/core"
)

// FtpScan FTP扫描函数（参考fscan设计）
func FtpScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 21 // FTP默认端口
	}

	// 优先检测匿名访问（类似fscan的FtpUnauth）
	if info.Username == "" && info.Password == "" {
		if err := ftpAnonymous(info); err == nil {
			fmt.Printf("[+] %s:%d ftp anonymous access\n", info.Host, info.Port)
			return nil // 发现匿名访问，停止进一步检测
		}
	}

	// 进行认证检测
	return ftpAuth(info)
}

// ftpAnonymous 检测匿名访问
func ftpAnonymous(info *core.HostInfo) error {
	// 获取context，如果没有则创建默认的
	ctx := info.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
	}

	timeout := info.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)

	// 创建带超时的context用于单个请求
	requestCtx, requestCancel := context.WithTimeout(ctx, timeout)
	defer requestCancel()

	// 先测试TCP连接
	var dialer net.Dialer
	conn, err := dialer.DialContext(requestCtx, "tcp", addr)
	if err != nil {
		return err
	}
	conn.Close()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 创建FTP连接
	ftpConn, err := ftp.Dial(addr, ftp.DialWithTimeout(timeout))
	if err != nil {
		return err
	}
	defer ftpConn.Quit()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 尝试匿名登录
	err = ftpConn.Login("anonymous", "anonymous@example.com")
	if err != nil {
		// 检查context是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 尝试另一种匿名登录方式
		err = ftpConn.Login("anonymous", "")
		if err != nil {
			// 检查context是否已取消
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// 尝试ftp用户
			err = ftpConn.Login("ftp", "")
			if err != nil {
				return err
			}
		}
	}

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 验证匿名访问权限 - 尝试列出目录
	_, err = ftpConn.List("/")
	return err
}

// ftpAuth 认证检测
func ftpAuth(info *core.HostInfo) error {
	// 获取context，如果没有则创建默认的
	ctx := info.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	timeout := info.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)

	// 创建带超时的context用于单个请求
	requestCtx, requestCancel := context.WithTimeout(ctx, timeout)
	defer requestCancel()

	// 先测试TCP连接
	var dialer net.Dialer
	conn, err := dialer.DialContext(requestCtx, "tcp", addr)
	if err != nil {
		return err
	}
	conn.Close()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 创建FTP连接 - 使用更短的超时
	ftpTimeout := timeout
	if ftpTimeout > 5*time.Second {
		ftpTimeout = 5 * time.Second // 强制限制FTP连接超时为5秒
	}

	ftpConn, err := ftp.Dial(addr, ftp.DialWithTimeout(ftpTimeout))
	if err != nil {
		return err
	}
	defer ftpConn.Quit()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 尝试登录
	err = ftpConn.Login(info.Username, info.Password)
	if err == nil {
		fmt.Printf("[+] %s:%d ftp %s:%s\n", info.Host, info.Port, info.Username, info.Password)
	}
	return err
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("ftp", FtpScan)
}
