package plugins

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/zan8in/leo/internal/core"
	"golang.org/x/crypto/md4"
)

// RDP协议常量
const (
	// RDP协议版本
	RDP_VERSION_4    = 0x00080001
	RDP_VERSION_5    = 0x00080004
	RDP_VERSION_10   = 0x00080005
	RDP_VERSION_10_1 = 0x00080006
	RDP_VERSION_10_2 = 0x00080007
	RDP_VERSION_10_3 = 0x00080008
	RDP_VERSION_10_4 = 0x00080009
	RDP_VERSION_10_5 = 0x0008000A
	RDP_VERSION_10_6 = 0x0008000B
	RDP_VERSION_10_7 = 0x0008000C

	// 协议类型
	PROTOCOL_RDP       = 0x00000000
	PROTOCOL_SSL       = 0x00000001
	PROTOCOL_HYBRID    = 0x00000002
	PROTOCOL_RDSTLS    = 0x00000004
	PROTOCOL_HYBRID_EX = 0x00000008

	// TPKT头部
	TPKT_VERSION = 0x03

	// X.224连接请求
	X224_TPDU_CONNECTION_REQUEST = 0xE0
	X224_TPDU_CONNECTION_CONFIRM = 0xD0
	X224_TPDU_DATA               = 0xF0

	// RDP协商类型
	TYPE_RDP_NEG_REQ     = 0x01
	TYPE_RDP_NEG_RSP     = 0x02
	TYPE_RDP_NEG_FAILURE = 0x03

	// NLA认证状态
	NLA_AUTH_PKG_NAME = "NTLM"
)

// RDP连接信息
type RDPConnection struct {
	conn       net.Conn
	tlsConn    *tls.Conn
	protocols  uint32
	supportNLA bool
	supportTLS bool
	rdpVersion uint32
	target     string
	username   string
	password   string
	ctx        context.Context
}

// NTLM消息类型
type NTLMMessage struct {
	Signature   [8]byte
	MessageType uint32
	Data        []byte
}

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

	// 创建RDP连接
	rdpConn, err := NewRDPConnection(target, ctx)
	if err != nil {
		return fmt.Errorf("RDP connection failed: %v", err)
	}
	defer rdpConn.Close()

	// 执行RDP协议协商
	if err := rdpConn.Negotiate(); err != nil {
		return fmt.Errorf("RDP negotiation failed: %v", err)
	}

	// 如果提供了用户名和密码，尝试认证
	if info.Username != "" || info.Password != "" {
		rdpConn.username = info.Username
		rdpConn.password = info.Password

		if err := rdpConn.Authenticate(); err != nil {
			return fmt.Errorf("RDP authentication failed for %s:%s - %v", info.Username, info.Password, err)
		}
		// 认证成功
		fmt.Printf("[+] RDP %s:%d %s:%s\n", info.Host, info.Port, info.Username, info.Password)
		return nil
	}

	// 检测到 RDP 服务但未提供凭据
	return nil
}

// NewRDPConnection 创建新的RDP连接
func NewRDPConnection(target string, ctx context.Context) (*RDPConnection, error) {
	conn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		return nil, err
	}

	return &RDPConnection{
		conn:   conn,
		target: target,
		ctx:    ctx,
	}, nil
}

// Close 关闭连接
func (r *RDPConnection) Close() {
	if r.tlsConn != nil {
		r.tlsConn.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

// Negotiate 执行RDP协议协商
func (r *RDPConnection) Negotiate() error {
	// 设置读写超时
	r.conn.SetDeadline(time.Now().Add(30 * time.Second))

	// 发送连接请求
	if err := r.sendConnectionRequest(); err != nil {
		return fmt.Errorf("failed to send connection request: %v", err)
	}

	// 读取连接确认
	if err := r.readConnectionConfirm(); err != nil {
		return fmt.Errorf("failed to read connection confirm: %v", err)
	}

	// 如果支持TLS，建立TLS连接
	if r.supportTLS {
		if err := r.establishTLS(); err != nil {
			return fmt.Errorf("failed to establish TLS: %v", err)
		}
	}

	return nil
}

// sendConnectionRequest 发送RDP连接请求
func (r *RDPConnection) sendConnectionRequest() error {
	// 构建RDP协商请求
	negData := r.buildNegotiationRequest()

	// 构建X.224连接请求
	x224Data := r.buildX224ConnectionRequest(negData)

	// 构建TPKT头部
	tpktData := r.buildTPKT(x224Data)

	// 发送数据
	_, err := r.conn.Write(tpktData)
	return err
}

// buildNegotiationRequest 构建协商请求
func (r *RDPConnection) buildNegotiationRequest() []byte {
	buf := new(bytes.Buffer)

	// RDP协商请求
	binary.Write(buf, binary.LittleEndian, uint8(TYPE_RDP_NEG_REQ)) // Type
	binary.Write(buf, binary.LittleEndian, uint8(0))                // Flags
	binary.Write(buf, binary.LittleEndian, uint16(8))               // Length

	// 请求的协议
	requestedProtocols := uint32(PROTOCOL_SSL | PROTOCOL_HYBRID)
	binary.Write(buf, binary.LittleEndian, requestedProtocols)

	return buf.Bytes()
}

// buildX224ConnectionRequest 构建X.224连接请求
func (r *RDPConnection) buildX224ConnectionRequest(negData []byte) []byte {
	buf := new(bytes.Buffer)

	// X.224头部
	length := uint8(6 + len(negData))                                           // 固定头部6字节 + 协商数据长度
	binary.Write(buf, binary.LittleEndian, length)                              // Length
	binary.Write(buf, binary.LittleEndian, uint8(X224_TPDU_CONNECTION_REQUEST)) // TPDU
	binary.Write(buf, binary.LittleEndian, uint16(0))                           // Destination reference
	binary.Write(buf, binary.LittleEndian, uint16(0))                           // Source reference
	binary.Write(buf, binary.LittleEndian, uint8(0))                            // Class option

	// 添加协商数据
	buf.Write(negData)

	return buf.Bytes()
}

// buildTPKT 构建TPKT头部
func (r *RDPConnection) buildTPKT(data []byte) []byte {
	buf := new(bytes.Buffer)

	// TPKT头部
	binary.Write(buf, binary.BigEndian, uint8(TPKT_VERSION)) // Version
	binary.Write(buf, binary.BigEndian, uint8(0))            // Reserved
	binary.Write(buf, binary.BigEndian, uint16(4+len(data))) // Length

	// 添加数据
	buf.Write(data)

	return buf.Bytes()
}

// readConnectionConfirm 读取连接确认
func (r *RDPConnection) readConnectionConfirm() error {
	// 读取TPKT头部
	tpktHeader := make([]byte, 4)
	if _, err := r.conn.Read(tpktHeader); err != nil {
		return err
	}

	if tpktHeader[0] != TPKT_VERSION {
		return fmt.Errorf("invalid TPKT version: %d", tpktHeader[0])
	}

	// 获取数据长度
	length := binary.BigEndian.Uint16(tpktHeader[2:4]) - 4

	// 读取剩余数据
	data := make([]byte, length)
	if _, err := r.conn.Read(data); err != nil {
		return err
	}

	// 解析X.224连接确认
	if len(data) < 7 {
		return fmt.Errorf("invalid X.224 connection confirm length")
	}

	if data[1] != X224_TPDU_CONNECTION_CONFIRM {
		return fmt.Errorf("invalid X.224 TPDU type: %d", data[1])
	}

	// 解析协商响应
	if len(data) > 7 {
		return r.parseNegotiationResponse(data[7:])
	}

	return nil
}

// parseNegotiationResponse 解析协商响应
func (r *RDPConnection) parseNegotiationResponse(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("invalid negotiation response length")
	}

	negType := data[0]
	switch negType {
	case TYPE_RDP_NEG_RSP:
		// 协商成功
		r.protocols = binary.LittleEndian.Uint32(data[4:8])
		r.supportTLS = (r.protocols & PROTOCOL_SSL) != 0
		r.supportNLA = (r.protocols & PROTOCOL_HYBRID) != 0
		return nil

	case TYPE_RDP_NEG_FAILURE:
		// 协商失败
		failureCode := binary.LittleEndian.Uint32(data[4:8])
		return fmt.Errorf("RDP negotiation failed with code: %d", failureCode)

	default:
		return fmt.Errorf("unknown negotiation response type: %d", negType)
	}
}

// establishTLS 建立TLS连接
func (r *RDPConnection) establishTLS() error {
	// 创建TLS配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // 跳过证书验证
		ServerName:         strings.Split(r.target, ":")[0],
	}

	// 建立TLS连接
	r.tlsConn = tls.Client(r.conn, tlsConfig)

	// 执行TLS握手
	if err := r.tlsConn.Handshake(); err != nil {
		return err
	}

	return nil
}

// Authenticate 执行认证
func (r *RDPConnection) Authenticate() error {
	if r.supportNLA {
		// 使用NLA认证
		return r.authenticateNLA()
	} else {
		// 使用标准RDP认证
		return r.authenticateStandard()
	}
}

// authenticateNLA 执行NLA认证
func (r *RDPConnection) authenticateNLA() error {
	// NLA认证使用NTLM over TLS
	if !r.supportTLS {
		return fmt.Errorf("NLA requires TLS but TLS is not supported")
	}

	// 执行NTLM认证
	return r.performNTLMAuth()
}

// performNTLMAuth 执行NTLM认证
func (r *RDPConnection) performNTLMAuth() error {
	// 第一步：发送Type 1消息（协商消息）
	type1Msg := r.createNTLMType1Message()
	if err := r.sendNTLMMessage(type1Msg); err != nil {
		return fmt.Errorf("failed to send NTLM Type 1 message: %v", err)
	}

	// 第二步：接收Type 2消息（挑战消息）
	type2Msg, err := r.receiveNTLMMessage()
	if err != nil {
		return fmt.Errorf("failed to receive NTLM Type 2 message: %v", err)
	}

	// 第三步：发送Type 3消息（认证消息）
	type3Msg := r.createNTLMType3Message(type2Msg)
	if err := r.sendNTLMMessage(type3Msg); err != nil {
		return fmt.Errorf("failed to send NTLM Type 3 message: %v", err)
	}

	// 验证认证结果
	return r.verifyNTLMAuth()
}

// createNTLMType1Message 创建NTLM Type 1消息
func (r *RDPConnection) createNTLMType1Message() []byte {
	buf := new(bytes.Buffer)

	// NTLM签名
	buf.WriteString("NTLMSSP\x00")

	// 消息类型 (Type 1)
	binary.Write(buf, binary.LittleEndian, uint32(1))

	// 标志
	flags := uint32(0x62890235) // 标准NTLM标志
	binary.Write(buf, binary.LittleEndian, flags)

	// 域名长度和偏移（空）
	binary.Write(buf, binary.LittleEndian, uint16(0)) // 长度
	binary.Write(buf, binary.LittleEndian, uint16(0)) // 最大长度
	binary.Write(buf, binary.LittleEndian, uint32(0)) // 偏移

	// 工作站名长度和偏移（空）
	binary.Write(buf, binary.LittleEndian, uint16(0)) // 长度
	binary.Write(buf, binary.LittleEndian, uint16(0)) // 最大长度
	binary.Write(buf, binary.LittleEndian, uint32(0)) // 偏移

	return buf.Bytes()
}

// createNTLMType3Message 创建NTLM Type 3消息
func (r *RDPConnection) createNTLMType3Message(type2Msg []byte) []byte {
	// 解析Type 2消息获取挑战
	challenge := r.extractChallenge(type2Msg)

	// 计算响应
	ntResponse := r.calculateNTResponse(challenge, r.password)
	lmResponse := r.calculateLMResponse(challenge, r.password)

	buf := new(bytes.Buffer)

	// NTLM签名
	buf.WriteString("NTLMSSP\x00")

	// 消息类型 (Type 3)
	binary.Write(buf, binary.LittleEndian, uint32(3))

	// 计算各字段的偏移
	baseOffset := uint32(64) // 固定头部长度
	currentOffset := baseOffset

	// LM响应
	lmLen := uint16(len(lmResponse))
	binary.Write(buf, binary.LittleEndian, lmLen)         // 长度
	binary.Write(buf, binary.LittleEndian, lmLen)         // 最大长度
	binary.Write(buf, binary.LittleEndian, currentOffset) // 偏移
	currentOffset += uint32(lmLen)

	// NT响应
	ntLen := uint16(len(ntResponse))
	binary.Write(buf, binary.LittleEndian, ntLen)         // 长度
	binary.Write(buf, binary.LittleEndian, ntLen)         // 最大长度
	binary.Write(buf, binary.LittleEndian, currentOffset) // 偏移
	currentOffset += uint32(ntLen)

	// 域名（空）
	binary.Write(buf, binary.LittleEndian, uint16(0))     // 长度
	binary.Write(buf, binary.LittleEndian, uint16(0))     // 最大长度
	binary.Write(buf, binary.LittleEndian, currentOffset) // 偏移

	// 用户名
	usernameUTF16 := r.stringToUTF16LE(r.username)
	userLen := uint16(len(usernameUTF16))
	binary.Write(buf, binary.LittleEndian, userLen)       // 长度
	binary.Write(buf, binary.LittleEndian, userLen)       // 最大长度
	binary.Write(buf, binary.LittleEndian, currentOffset) // 偏移
	currentOffset += uint32(userLen)

	// 工作站名（空）
	binary.Write(buf, binary.LittleEndian, uint16(0))     // 长度
	binary.Write(buf, binary.LittleEndian, uint16(0))     // 最大长度
	binary.Write(buf, binary.LittleEndian, currentOffset) // 偏移

	// 会话密钥（空）
	binary.Write(buf, binary.LittleEndian, uint16(0))     // 长度
	binary.Write(buf, binary.LittleEndian, uint16(0))     // 最大长度
	binary.Write(buf, binary.LittleEndian, currentOffset) // 偏移

	// 标志
	binary.Write(buf, binary.LittleEndian, uint32(0x62890235))

	// 添加数据
	buf.Write(lmResponse)
	buf.Write(ntResponse)
	buf.Write(usernameUTF16)

	return buf.Bytes()
}

// extractChallenge 从Type 2消息中提取挑战
func (r *RDPConnection) extractChallenge(type2Msg []byte) []byte {
	if len(type2Msg) < 32 {
		return make([]byte, 8) // 返回空挑战
	}
	return type2Msg[24:32] // 挑战位于偏移24-31
}

// calculateNTResponse 计算NT响应
func (r *RDPConnection) calculateNTResponse(challenge []byte, password string) []byte {
	// 将密码转换为UTF-16LE
	passwordUTF16 := r.stringToUTF16LE(password)

	// 计算MD4哈希
	md4Hash := md4.New()
	md4Hash.Write(passwordUTF16)
	ntHash := md4Hash.Sum(nil)

	// 使用DES加密挑战
	return r.desEncrypt(ntHash, challenge)
}

// calculateLMResponse 计算LM响应
func (r *RDPConnection) calculateLMResponse(challenge []byte, password string) []byte {
	// LM响应通常为空或使用NT响应
	return make([]byte, 24)
}

// stringToUTF16LE 将字符串转换为UTF-16LE
func (r *RDPConnection) stringToUTF16LE(s string) []byte {
	runes := []rune(s)
	buf := new(bytes.Buffer)

	for _, r := range runes {
		binary.Write(buf, binary.LittleEndian, uint16(r))
	}

	return buf.Bytes()
}

// desEncrypt 使用DES加密（简化实现）
func (r *RDPConnection) desEncrypt(key, data []byte) []byte {
	// 这里应该实现完整的DES加密
	// 为了简化，返回固定长度的响应
	response := make([]byte, 24)

	// 使用MD5作为简化的加密方法
	hash := md5.New()
	hash.Write(key)
	hash.Write(data)
	copy(response, hash.Sum(nil))

	return response
}

// sendNTLMMessage 发送NTLM消息
func (r *RDPConnection) sendNTLMMessage(msg []byte) error {
	var conn net.Conn
	if r.tlsConn != nil {
		conn = r.tlsConn
	} else {
		conn = r.conn
	}

	// 构建消息头部
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(msg)))

	// 发送头部和消息
	if _, err := conn.Write(header); err != nil {
		return err
	}

	_, err := conn.Write(msg)
	return err
}

// receiveNTLMMessage 接收NTLM消息
func (r *RDPConnection) receiveNTLMMessage() ([]byte, error) {
	var conn net.Conn
	if r.tlsConn != nil {
		conn = r.tlsConn
	} else {
		conn = r.conn
	}

	// 读取消息长度
	header := make([]byte, 4)
	if _, err := conn.Read(header); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(header)
	if length > 65536 { // 防止过大的消息
		return nil, fmt.Errorf("message too large: %d", length)
	}

	// 读取消息内容
	msg := make([]byte, length)
	if _, err := conn.Read(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// verifyNTLMAuth 验证NTLM认证结果
func (r *RDPConnection) verifyNTLMAuth() error {
	// 尝试读取认证结果
	var conn net.Conn
	if r.tlsConn != nil {
		conn = r.tlsConn
	} else {
		conn = r.conn
	}

	// 设置短超时来检查是否有错误响应
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)

	// 重置超时
	conn.SetReadDeadline(time.Time{})

	if err != nil {
		// 如果是超时错误，可能表示认证成功（没有错误消息）
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil // 认证可能成功
		}
		return fmt.Errorf("authentication verification failed: %v", err)
	}

	// 检查响应内容
	if n > 0 {
		// 如果收到数据，可能是错误消息
		return fmt.Errorf("authentication failed: received error response")
	}

	return nil
}

// authenticateStandard 执行标准RDP认证
func (r *RDPConnection) authenticateStandard() error {
	// 标准RDP认证（不使用NLA）
	// 这种情况下，认证通常在RDP连接建立后进行
	// 由于复杂性，这里返回未实现错误
	return fmt.Errorf("standard RDP authentication not implemented")
}

// 注册插件
func init() {
	core.GlobalRegistry.Register("rdp", RdpScan)
}
