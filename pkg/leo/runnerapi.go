package leo

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
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

	options.Users = append(options.Users, Userdict[options.Service]...)
	options.Passwords = append(options.Passwords, Passdict...)

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
			pass := GetPasswordByUser(username, password)
			host := hostinfo.HostInfo.Host
			m := hostinfo.Model

			if err := runner.execute.start(host, username, pass, m); err == nil {
				gologger.Print().Msgf("[+]Find success, host: %s, username: %s, password: %s", host, username, pass)
				resultChan <- &RunnerApiHostInfo{HostInfo: hostinfo.HostInfo, Username: username, Password: pass}
			} else {
				// fmt.Println("host:", host, "username:", username, "password:", pass, "err:", err)
			}

		})
		defer p.Release()

		for _, host := range runner.options.Hosts {
			if m, _, err := runner.execute.validateService(host.Host, host.Port); err != nil && m == nil {
				gologger.Error().Msgf("host: %s, port: %s, err: %s", host.Host, host.Port, err)
				continue
			} else {
				// 先验证端口存活
				alive, err := IsAliveWithRetries(host.Host, host.Port, runner.options.Retries, 6*time.Second)
				if !alive && err != nil {
					gologger.Print().Msgf("%s", err.Error())
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

var (
	Userdict = map[string][]string{
		"ftp":      {"ftp", "admin", "www", "web", "root", "db", "wwwroot", "data"},
		"mysql":    {"root", "mysql"},
		"mssql":    {"sa", "sql"},
		"smb":      {"administrator", "admin", "guest"},
		"rdp":      {"administrator", "admin", "guest"},
		"postgres": {"postgres", "admin"},
		"ssh":      {"root", "admin"},
		"mongodb":  {"root", "admin"},
		"oracle":   {"sys", "system", "admin", "test", "web", "orcl"},
	}

	Passdict = []string{"123456", "admin", "admin123", "root", "", "pass123", "pass@123", "password", "123123", "654321", "111111", "123", "1", "admin@123", "Admin@123", "admin123!@#", "{user}", "{user}1", "{user}111", "{user}123", "{user}@123", "{user}_123", "{user}#123", "{user}@111", "{user}@2019", "{user}@123#4", "P@ssw0rd!", "P@ssw0rd", "Passw0rd", "qwe123", "12345678", "test", "test123", "123qwe", "123qwe!@#", "123456789", "123321", "666666", "a123456.", "123456~a", "123456!a", "000000", "1234567890", "8888888", "!QAZ2wsx", "1qaz2wsx", "abc123", "abc123456", "1qaz@WSX", "a11111", "a12345", "Aa1234", "Aa1234.", "Aa12345", "a123456", "a123123", "Aa123123", "Aa123456", "Aa12345.", "sysadmin", "system", "1qaz!QAZ", "2wsx@WSX", "qwe123!@#", "Aa123456!", "A123456s!", "sa123456", "1q2w3e", "Charge123", "Aa123456789"}
)

func GetPasswordByUser(user, pass string) string {
	if strings.HasPrefix(pass, "{user}") {
		pass = strings.ReplaceAll(pass, "{user}", user)
	}
	return pass

}
