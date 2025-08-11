package plugins

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/zan8in/leo/internal/core"
	"golang.org/x/crypto/ssh"
)

// SshScan SSH扫描函数（参考fscan设计）
func SshScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 22 // SSH默认端口
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

	// 进行SSH认证
	return sshAuth(requestCtx, info)
}

// sshAuth SSH认证函数
func sshAuth(ctx context.Context, info *core.HostInfo) error {
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

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)

	// 创建SSH客户端配置
	config := &ssh.ClientConfig{
		User: info.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(info.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略主机密钥验证（仅用于扫描）
		Timeout:         timeout,
	}

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 尝试SSH连接
	return trySshConnect(ctx, addr, config, info)
}

// trySshConnect 尝试SSH连接
func trySshConnect(ctx context.Context, addr string, config *ssh.ClientConfig, info *core.HostInfo) error {
	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 创建带超时的TCP连接
	dialer := &net.Dialer{
		Timeout: config.Timeout,
	}

	// 使用context进行TCP连接
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 创建SSH连接
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return err
	}
	defer sshConn.Close()

	// 创建SSH客户端
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 尝试创建会话来验证连接
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 执行简单命令验证权限
	err = session.Run("echo 'ssh_test'")
	if err == nil {
		// 认证成功，输出结果
		fmt.Printf("[+] %s:%d ssh %s:%s\n", info.Host, info.Port, info.Username, info.Password)
	}

	return err
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("ssh", SshScan)
}
