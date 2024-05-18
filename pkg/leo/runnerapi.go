package leo

import (
	"fmt"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
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
			if m, ret, err := runner.execute.validateService(host.Host, host.Port); err != nil {
				fmt.Println("m:", m, "ret:", ret, "err:", err)
				continue
			} else {
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

	p.Release()
	return <-resultChan
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
