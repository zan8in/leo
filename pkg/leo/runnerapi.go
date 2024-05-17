package leo

import (
	"fmt"
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

type RunnerApiResult struct {
	HostInfo HostInfo
	Username string
	Password string
}

func (runner *Runner) RunApi() *RunnerApiResult {

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
					return &RunnerApiResult{HostInfo: host, Username: username, Password: pass}
				} else {
					fmt.Println("host:", host, "username:", username, "password:", pass, "err:", err)
				}
			}
		}
	}

	return nil
}
