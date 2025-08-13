package plugins

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/zan8in/leo/internal/core"
)

// Telnet协议常量
const (
	// Telnet命令
	TELNET_IAC  = 255 // Interpret As Command
	TELNET_DONT = 254 // Don't
	TELNET_DO   = 253 // Do
	TELNET_WONT = 252 // Won't
	TELNET_WILL = 251 // Will
	TELNET_SB   = 250 // Subnegotiation Begin
	TELNET_SE   = 240 // Subnegotiation End

	// Telnet选项
	TELNET_ECHO                = 1  // Echo
	TELNET_SUPPRESS_GO_AHEAD   = 3  // Suppress Go Ahead
	TELNET_TERMINAL_TYPE       = 24 // Terminal Type
	TELNET_WINDOW_SIZE         = 31 // Window Size
	TELNET_TERMINAL_SPEED      = 32 // Terminal Speed
	TELNET_REMOTE_FLOW_CONTROL = 33 // Remote Flow Control
	TELNET_LINEMODE            = 34 // Linemode
	TELNET_ENVIRONMENT         = 36 // Environment
)

// TelnetScan Telnet弱口令扫描函数
func TelnetScan(info *core.HostInfo) error {
	if info.Port == 0 {
		info.Port = 23 // Telnet默认端口
	}

	// 获取context，如果没有则创建默认的
	ctx := info.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}

	// 使用超时时间，如果为0则使用默认值
	timeout := info.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
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

	// 建立TCP连接
	target := fmt.Sprintf("%s:%d", info.Host, info.Port)
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.DialContext(requestCtx, "tcp", target)
	if err != nil {
		return fmt.Errorf("telnet connection failed: %v", err)
	}
	defer conn.Close()

	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 创建Telnet客户端
	telnetClient := &TelnetClient{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		ctx:      requestCtx,
		timeout:  timeout,
		username: info.Username,
		password: info.Password,
	}

	// 执行Telnet认证
	if err := telnetClient.Authenticate(); err != nil {
		return fmt.Errorf("telnet authentication failed for %s:%s - %v", info.Username, info.Password, err)
	}

	// 认证成功
	fmt.Printf("[+] %s:%d telnet %s:%s\n", info.Host, info.Port, info.Username, info.Password)
	return nil
}

// TelnetClient Telnet客户端结构
type TelnetClient struct {
	conn     net.Conn
	reader   *bufio.Reader
	ctx      context.Context
	timeout  time.Duration
	username string
	password string
}

// Authenticate 执行Telnet认证
func (t *TelnetClient) Authenticate() error {
	// 设置连接超时
	t.conn.SetDeadline(time.Now().Add(t.timeout))

	// 处理初始Telnet协商
	if err := t.handleTelnetNegotiation(); err != nil {
		return fmt.Errorf("telnet negotiation failed: %v", err)
	}

	// 等待登录提示
	if err := t.waitForLoginPrompt(); err != nil {
		return fmt.Errorf("failed to get login prompt: %v", err)
	}

	// 发送用户名
	if err := t.sendUsername(); err != nil {
		return fmt.Errorf("failed to send username: %v", err)
	}

	// 等待密码提示
	if err := t.waitForPasswordPrompt(); err != nil {
		return fmt.Errorf("failed to get password prompt: %v", err)
	}

	// 发送密码
	if err := t.sendPassword(); err != nil {
		return fmt.Errorf("failed to send password: %v", err)
	}

	// 验证登录是否成功
	if err := t.verifyLogin(); err != nil {
		return fmt.Errorf("login verification failed: %v", err)
	}

	return nil
}

// handleTelnetNegotiation 处理Telnet协议协商
func (t *TelnetClient) handleTelnetNegotiation() error {
	// 读取并处理初始的Telnet协商命令
	for i := 0; i < 10; i++ { // 最多处理10轮协商
		// 检查context
		select {
		case <-t.ctx.Done():
			return t.ctx.Err()
		default:
		}

		// 设置短超时读取
		t.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		byte1, err := t.reader.ReadByte()
		if err != nil {
			// 如果是超时，可能协商已完成
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return err
		}

		// 如果不是IAC命令，可能是登录提示的开始
		if byte1 != TELNET_IAC {
			// 将字节放回缓冲区
			t.reader = bufio.NewReader(io.MultiReader(bytes.NewReader([]byte{byte1}), t.reader))
			break
		}

		// 读取命令和选项
		cmd, err := t.reader.ReadByte()
		if err != nil {
			return err
		}

		option, err := t.reader.ReadByte()
		if err != nil {
			return err
		}

		// 处理Telnet命令
		if err := t.handleTelnetCommand(cmd, option); err != nil {
			return err
		}
	}

	// 重置读取超时
	t.conn.SetReadDeadline(time.Now().Add(t.timeout))
	return nil
}

// handleTelnetCommand 处理Telnet命令
func (t *TelnetClient) handleTelnetCommand(cmd, option byte) error {
	switch cmd {
	case TELNET_DO:
		// 服务器要求我们启用某个选项，我们拒绝
		return t.sendTelnetResponse(TELNET_WONT, option)
	case TELNET_DONT:
		// 服务器要求我们不要使用某个选项，我们同意
		return nil
	case TELNET_WILL:
		// 服务器将启用某个选项
		if option == TELNET_ECHO || option == TELNET_SUPPRESS_GO_AHEAD {
			// 同意服务器启用回显和抑制前进
			return t.sendTelnetResponse(TELNET_DO, option)
		} else {
			// 其他选项我们不需要
			return t.sendTelnetResponse(TELNET_DONT, option)
		}
	case TELNET_WONT:
		// 服务器不会启用某个选项
		return nil
	case TELNET_SB:
		// 子协商，读取到SE为止
		return t.skipSubnegotiation()
	}
	return nil
}

// sendTelnetResponse 发送Telnet响应
func (t *TelnetClient) sendTelnetResponse(cmd, option byte) error {
	response := []byte{TELNET_IAC, cmd, option}
	_, err := t.conn.Write(response)
	return err
}

// skipSubnegotiation 跳过子协商
func (t *TelnetClient) skipSubnegotiation() error {
	for {
		byte1, err := t.reader.ReadByte()
		if err != nil {
			return err
		}
		if byte1 == TELNET_IAC {
			byte2, err := t.reader.ReadByte()
			if err != nil {
				return err
			}
			if byte2 == TELNET_SE {
				break
			}
		}
	}
	return nil
}

// waitForLoginPrompt 等待登录提示
func (t *TelnetClient) waitForLoginPrompt() error {
	promptPatterns := []string{
		"login:",
		"username:",
		"user:",
		"账号:",
		"用户名:",
		"登录:",
	}

	return t.waitForPrompt(promptPatterns, 15*time.Second)
}

// waitForPasswordPrompt 等待密码提示
func (t *TelnetClient) waitForPasswordPrompt() error {
	promptPatterns := []string{
		"password:",
		"passwd:",
		"密码:",
		"口令:",
	}

	return t.waitForPrompt(promptPatterns, 10*time.Second)
}

// waitForPrompt 等待指定的提示符
func (t *TelnetClient) waitForPrompt(patterns []string, timeout time.Duration) error {
	// 设置读取超时
	t.conn.SetReadDeadline(time.Now().Add(timeout))
	defer t.conn.SetReadDeadline(time.Now().Add(t.timeout))

	buffer := make([]byte, 0, 1024)
	for {
		// 检查context
		select {
		case <-t.ctx.Done():
			return t.ctx.Err()
		default:
		}

		// 读取一个字节
		b, err := t.reader.ReadByte()
		if err != nil {
			return err
		}

		buffer = append(buffer, b)

		// 保持缓冲区大小合理
		if len(buffer) > 2048 {
			buffer = buffer[1024:] // 保留后半部分
		}

		// 转换为小写字符串进行匹配
		text := strings.ToLower(string(buffer))

		// 检查是否匹配任何提示符模式
		for _, pattern := range patterns {
			if strings.Contains(text, strings.ToLower(pattern)) {
				return nil
			}
		}

		// 如果缓冲区包含错误信息，提前返回
		if strings.Contains(text, "connection refused") ||
			strings.Contains(text, "connection closed") ||
			strings.Contains(text, "access denied") {
			return fmt.Errorf("connection error detected")
		}
	}
}

// sendUsername 发送用户名
func (t *TelnetClient) sendUsername() error {
	// 检查context
	select {
	case <-t.ctx.Done():
		return t.ctx.Err()
	default:
	}

	data := t.username + "\r\n"
	_, err := t.conn.Write([]byte(data))
	return err
}

// sendPassword 发送密码
func (t *TelnetClient) sendPassword() error {
	// 检查context
	select {
	case <-t.ctx.Done():
		return t.ctx.Err()
	default:
	}

	data := t.password + "\r\n"
	_, err := t.conn.Write([]byte(data))
	return err
}

// verifyLogin 验证登录是否成功
func (t *TelnetClient) verifyLogin() error {
	// 设置读取超时
	t.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer t.conn.SetReadDeadline(time.Now().Add(t.timeout))

	buffer := make([]byte, 0, 2048)
	startTime := time.Now()

	for time.Since(startTime) < 8*time.Second {
		// 检查context
		select {
		case <-t.ctx.Done():
			return t.ctx.Err()
		default:
		}

		// 尝试读取数据
		t.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		b, err := t.reader.ReadByte()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时，继续尝试
				continue
			}
			return err
		}

		buffer = append(buffer, b)

		// 保持缓冲区大小合理
		if len(buffer) > 4096 {
			buffer = buffer[2048:] // 保留后半部分
		}

		// 转换为小写字符串进行分析
		text := strings.ToLower(string(buffer))

		// 检查登录失败的标志
		failurePatterns := []string{
			"login incorrect",
			"login failed",
			"authentication failed",
			"access denied",
			"invalid",
			"wrong",
			"error",
			"failed",
			"incorrect",
			"denied",
			"登录失败",
			"认证失败",
			"用户名或密码错误",
			"访问被拒绝",
		}

		for _, pattern := range failurePatterns {
			if strings.Contains(text, pattern) {
				return fmt.Errorf("authentication failed: %s", pattern)
			}
		}

		// 检查登录成功的标志
		successPatterns := []string{
			"$", // Shell提示符
			"#", // Root提示符
			">", // Windows命令提示符
			"welcome",
			"last login",
			"successful",
			"欢迎",
			"成功",
		}

		// 检查是否包含成功标志，并且没有再次出现登录提示
		hasSuccessPattern := false
		for _, pattern := range successPatterns {
			if strings.Contains(text, pattern) {
				hasSuccessPattern = true
				break
			}
		}

		// 如果有成功标志且没有再次出现登录提示，认为登录成功
		if hasSuccessPattern && !strings.Contains(text, "login:") && !strings.Contains(text, "username:") {
			return nil
		}

		// 如果缓冲区足够大且包含提示符，可能登录成功
		if len(buffer) > 100 {
			// 检查最后几个字符是否像提示符
			lastChars := string(buffer[len(buffer)-10:])
			if strings.Contains(lastChars, "$") || strings.Contains(lastChars, "#") || strings.Contains(lastChars, ">") {
				return nil
			}
		}
	}

	// 如果没有明确的失败信息，且读取到了数据，可能是成功的
	if len(buffer) > 50 {
		return nil
	}

	return fmt.Errorf("login verification timeout or failed")
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("telnet", TelnetScan)
}
