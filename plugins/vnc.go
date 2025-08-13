package plugins

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mitchellh/go-vnc"
	"github.com/zan8in/leo/internal/core"
)

// VncScan VNC弱口令扫描函数
func VncScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 5900 // VNC默认端口
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

	// 如果有指定密码，直接尝试该密码
	if info.Password != "" {
		success, sessionInfo, err := VncConn(ctx, info, info.Password)
		if success {
			fmt.Printf("[+] %s:%d vnc %s %s\n", info.Host, info.Port, info.Password, sessionInfo)
		}
		return err
	}

	// 尝试空密码认证
	success, sessionInfo, err := VncConn(ctx, info, "")
	if success {
		fmt.Printf("[+] %s:%d vnc empty password %s\n", info.Host, info.Port, sessionInfo)
		return nil
	}

	return err
}

// VncConn 尝试建立VNC连接并验证会话
func VncConn(ctx context.Context, info *core.HostInfo, pass string) (bool, string, error) {
	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return false, "", ctx.Err()
	default:
	}

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)

	// 创建带超时的TCP连接
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, "", err
	}
	defer conn.Close()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return false, "", ctx.Err()
	default:
	}

	// 如果密码为空，先尝试无认证连接
	if pass == "" {
		// 尝试无认证连接
		vncConn, err := vnc.Client(conn, &vnc.ClientConfig{})
		if err == nil {
			defer vncConn.Close()
			// 验证会话信息
			sessionInfo := validateVncSession(vncConn)
			if sessionInfo != "" {
				return true, fmt.Sprintf("(unauthorized access) %s", sessionInfo), nil
			}
		}

		// 无认证失败，检查错误类型
		if strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), "security") {
			// 这是正常的认证要求，重新建立连接尝试空密码
			conn.Close()
			conn, err = dialer.DialContext(ctx, "tcp", addr)
			if err != nil {
				return false, "", err
			}
			defer conn.Close()

			// 尝试空密码认证
			config := &vnc.ClientConfig{
				Auth: []vnc.ClientAuth{
					&vnc.PasswordAuth{Password: ""},
				},
			}

			vncConn, err := vnc.Client(conn, config)
			if err != nil {
				return false, "", err
			}
			defer vncConn.Close()

			// 验证会话信息
			sessionInfo := validateVncSession(vncConn)
			if sessionInfo != "" {
				return true, sessionInfo, nil
			}
			return false, "", fmt.Errorf("VNC session validation failed")
		}

		// 其他类型的错误（连接问题等）
		return false, "", err
	}

	// 使用指定密码进行认证
	config := &vnc.ClientConfig{
		Auth: []vnc.ClientAuth{
			&vnc.PasswordAuth{Password: pass},
		},
	}

	vncConn, err := vnc.Client(conn, config)
	if err != nil {
		return false, "", err
	}
	defer vncConn.Close()

	// 验证会话信息
	sessionInfo := validateVncSession(vncConn)
	if sessionInfo != "" {
		return true, sessionInfo, nil
	}

	return false, "", fmt.Errorf("VNC session validation failed")
}

// validateVncSession 验证VNC会话并获取会话信息
func validateVncSession(conn *vnc.ClientConn) string {
	if conn == nil {
		return ""
	}

	// 获取桌面信息
	desktopName := conn.DesktopName
	width := conn.FrameBufferWidth
	height := conn.FrameBufferHeight

	// 尝试请求帧缓冲区更新来验证连接有效性
	err := conn.FramebufferUpdateRequest(false, 0, 0, 1, 1)
	if err != nil {
		return ""
	}

	// 构建会话信息字符串
	sessionInfo := fmt.Sprintf("desktop:'%s' resolution:%dx%d", desktopName, width, height)
	
	// 如果桌面名称为空，使用默认描述
	if desktopName == "" {
		sessionInfo = fmt.Sprintf("resolution:%dx%d", width, height)
	}

	return sessionInfo
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("vnc", VncScan)
}
