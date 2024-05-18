package leo

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	portScan "github.com/XinRoom/go-portScan/core/port"
	tcp "github.com/XinRoom/go-portScan/core/port/tcp"
	"github.com/panjf2000/ants/v2"
	"github.com/zan8in/gologger"
)

func NewRunnerApi(options *Options) (*Runner, error) {
	runner := &Runner{
		options: options,
	}

	defaultPort := DefaultServicePort[options.Service]

	if len(options.User) == 0 && len(options.UserFile) == 0 {
		options.Users = append(options.Users, defaultPort.Users...)
	} else {
		options.convertUsers()
	}

	if len(options.Users) == 0 {
		return runner, ErrNoUsers
	}

	options.Passwords = initPasswords()

	if len(options.Password) == 0 && len(options.PasswordFile) == 0 {
		options.Passwords = append(options.Passwords, defaultPort.Passwords...)
	} else {
		options.convertPasswords()
	}

	if len(options.Passwords) == 0 {
		return runner, ErrNoPasses
	}

	options.Count = uint32(len(options.Hosts) * len(options.Users) * len(options.Passwords))

	runner.execute = NewExecute(options)

	fmt.Println("count:", options.Count, "users:", len(options.Users), "passwords:", len(options.Passwords), "hosts:", options.Hosts, "service:", options.Service)

	return runner, nil
}

type RunnerApiHostInfo struct {
	HostInfo HostInfo
	Username string
	Password string
	Model    any
}

func (runner *Runner) RunApi() *RunnerApiHostInfo {

	var resultChan = make(chan *RunnerApiHostInfo, 1)
	var p *ants.PoolWithFunc

	go func() {
		var wg sync.WaitGroup
		ticker := time.NewTicker(time.Second / time.Duration(runner.options.RateLimit))
		p, _ = ants.NewPoolWithFunc(runner.options.Concurrency, func(p any) {
			defer wg.Done()
			<-ticker.C

			hostinfo := p.(*RunnerApiHostInfo)
			username, password := hostinfo.Username, hostinfo.Password
			pass := handlePassword(username, password)
			host := hostinfo.HostInfo.Host
			m := hostinfo.Model

			if err := runner.execute.start(host, username, pass, m); err == nil {
				resultChan <- &RunnerApiHostInfo{HostInfo: hostinfo.HostInfo, Username: username, Password: pass}
			} else {
				fmt.Println("host:", host, "username:", username, "password:", pass, "err:", err)
			}

		})
		defer p.Release()

		for _, host := range runner.options.Hosts {
			if m, _, err := runner.execute.validateService(host.Host, host.Port); err != nil {
				gologger.Error().Msgf("host: %s, port: %s, err: %s", host.Host, host.Port, err)
				continue
			} else {
				// 先验证端口存活
				alive, err := IsAliveWithRetries(host.Host, host.Port, runner.options.Retries, 6*time.Second)
				if !alive && err != nil {
					gologger.Error().Msgf("%s", err.Error())
					continue
				}
				// 如果端口存活，再进行爆破
				for _, username := range runner.options.Users {
					for _, password := range runner.options.Passwords {
						wg.Add(1)
						_ = p.Invoke(&RunnerApiHostInfo{HostInfo: host, Username: username, Password: password, Model: m})

					}
				}
			}
		}
		wg.Wait()

		close(resultChan)
	}()

	select {
	case result := <-resultChan:
		if p != nil {
			p.Release()
		}
		return result
	case <-time.After(24 * time.Hour):
		return nil
	}
}

func (runner *Runner) RunApi2() *RunnerApiHostInfo {

	for _, host := range runner.options.Hosts {
		m, ret, err := runner.execute.validateService(host.Host, host.Port)
		if err != nil {
			fmt.Println("m:", m, "ret:", ret, "err:", err)
			continue
		}
		for _, username := range runner.options.Users {
			for _, password := range runner.options.Passwords {
				pass := handlePassword(username, password)
				if err := runner.execute.start(host.Host, username, pass, m); err == nil {
					return &RunnerApiHostInfo{HostInfo: host, Username: username, Password: pass}
				} else {
					fmt.Println("host:", host, "username:", username, "password:", pass, "err:", err)
				}
			}
		}
	}

	return nil
}

// IsAliveWithRetries 尝试连接指定的 IP 地址和端口，并包含重试逻辑
func IsAliveWithRetries(ip string, port string, retries int, timeout time.Duration) (bool, error) {

	retChan := make(chan portScan.OpenIpPort, 1)

	ss, err := tcp.NewTcpScanner(retChan, tcp.DefaultTcpOption)
	if err != nil {
		return false, err
	}

	if iprst := net.ParseIP(ip); iprst != nil {
		if portrst, err := stringToUint16(port); err == nil {
			ss.Scan(iprst, portrst)
			ss.Wait()
		}
	}

	if len(retChan) == 0 {
		close(retChan)
		return false, fmt.Errorf("%s:%s is not open", ip, port)
	}

	select {
	case <-retChan:
		return true, nil
	case <-time.After(time.Duration(30) * time.Second):
		return false, fmt.Errorf("timeout %s:%s is not open", ip, port)
	}

}

func stringToUint16(s string) (uint16, error) {
	// 首先尝试将字符串解析为int64
	i, err := strconv.ParseInt(s, 10, 16) // 第三个参数指定了解析的整数类型的大小，这里我们指定为16位（即uint16的范围）
	if err != nil {
		return 0, err // 返回错误
	}

	// 检查转换后的整数是否在uint16的范围内
	if i < 0 || i > math.MaxUint16 { // 注意：需要导入"math"包来使用MaxUint16
		return 0, fmt.Errorf("value %d out of range for uint16", i)
	}

	// 将int64转换为uint16
	return uint16(i), nil
}

func IsAliveWithRetries2(ip string, port string, retries int, timeout time.Duration) (bool, error) {
	target := fmt.Sprintf("%s:%s", ip, port)
	var err error
	for i := 0; i < retries; i++ {
		conn, err := net.DialTimeout("tcp", target, timeout)
		if err == nil {
			// 连接成功，关闭连接并返回 true
			conn.Close()
			return true, nil
		}
		// 如果连接失败，等待一段时间后重试（可选的退避策略）
		if i < retries-1 {
			// 这里可以添加退避策略，比如指数退避
			time.Sleep(time.Duration(i) * time.Second) // 示例：线性退避
		}
	}
	// 所有重试都失败，返回 false 和最后一个错误
	return false, err
}
