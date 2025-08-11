package plugins

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/zan8in/leo/internal/core"
)

// RdpScan RDP弱口令扫描插件
func RdpScan(info *core.HostInfo) error {
	// 从 info.Context 获取上下文，如果没有则创建默认超时上下文
	ctx := info.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), info.Timeout*3)
		defer cancel()
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 构建目标地址
	target := fmt.Sprintf("%s:%d", info.Host, info.Port)

	// 尝试连接 RDP 端口
	conn, err := net.DialTimeout("tcp", target, info.Timeout)
	if err != nil {
		return fmt.Errorf("RDP connection failed: %v", err)
	}
	defer conn.Close()

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// RDP 协议握手检测
	if err := rdpHandshake(conn, ctx); err != nil {
		return fmt.Errorf("RDP handshake failed: %v", err)
	}

	// 如果提供了用户名和密码，尝试认证
	if info.Username != "" || info.Password != "" {
		if err := rdpAuth(conn, info.Username, info.Password, ctx); err != nil {
			return fmt.Errorf("RDP authentication failed for %s:%s - %v", info.Username, info.Password, err)
		}
		// 认证成功
		fmt.Printf("[+] RDP %s:%d %s:%s\n", info.Host, info.Port, info.Username, info.Password)
		return nil
	}

	// 检测到 RDP 服务但未提供凭据
	// fmt.Printf("[*] RDP service detected on %s:%d\n", info.Host, info.Port)
	return nil
}

// rdpHandshake 执行 RDP 协议握手
func rdpHandshake(conn net.Conn, ctx context.Context) error {
	// 设置读写超时
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// 发送 RDP 连接请求 (简化版)
	// 这里使用基本的 RDP 连接请求包
	rdpRequest := []byte{
		0x03, 0x00, 0x00, 0x13, // TPKT Header
		0x0e, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	// 检查上下文
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 发送请求
	if _, err := conn.Write(rdpRequest); err != nil {
		return fmt.Errorf("failed to send RDP request: %v", err)
	}

	// 读取响应
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read RDP response: %v", err)
	}

	// 简单验证响应是否为有效的 RDP 响应
	if n < 4 || buffer[0] != 0x03 {
		return fmt.Errorf("invalid RDP response")
	}

	return nil
}

// rdpAuth 执行 RDP 认证 (简化版)
func rdpAuth(conn net.Conn, username, password string, ctx context.Context) error {
	// 检查上下文
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 注意：这里是简化的实现
	// 实际的 RDP 认证需要完整的协议实现，包括 TLS 握手、认证协商等
	// 在生产环境中，建议使用专门的 RDP 库如 go-rdp

	// 模拟认证过程
	time.Sleep(100 * time.Millisecond) // 模拟网络延迟

	// 这里可以根据实际需求实现完整的 RDP 认证逻辑
	// 目前返回错误表示认证失败，实际使用时需要实现真正的认证
	return fmt.Errorf("RDP authentication not fully implemented")
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("rdp", RdpScan)
}
