package leo

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/panjf2000/ants"
	"github.com/zan8in/gologger"
	"github.com/zan8in/leo/pkg/ssh"
)

var Ticker *time.Ticker

type Runner struct {
	options *Options
}

type TargetInfo struct {
	username string
	password string
}

func NewRunner(options *Options) (*Runner, error) {
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

	if len(options.Password) == 0 && len(options.PasswordFile) == 0 {
		options.Passwords = append(options.Passwords, defaultPort.Passwords...)
	} else {
		options.convertPasswords()
	}

	if len(options.Passwords) == 0 {
		return runner, ErrNoPasses
	}

	options.showBanner()

	// whitespace show banner
	fmt.Println()

	return runner, nil
}

func (runner *Runner) Run() error {

	m, err := runner.validateService()
	if err != nil {
		return err
	}

	Ticker = time.NewTicker(time.Second / time.Duration(runner.options.RateLimit))
	var wg sync.WaitGroup

	p, _ := ants.NewPoolWithFunc(runner.options.Concurrency, func(p any) {
		defer wg.Done()
		<-Ticker.C

		targetInfo := p.(*TargetInfo)

		err := runner.start(targetInfo.username, targetInfo.password, m)
		if err != nil {
			gologger.Debug().Msgf("%s:%s %s", targetInfo.username, targetInfo.password, err.Error())
			return
		}

		gologger.Print().Msgf("%s:%s successed!", targetInfo.username, targetInfo.password)
	})
	defer p.Release()

	for _, username := range runner.options.Users {
		for _, password := range runner.options.Passwords {
			wg.Add(1)
			p.Invoke(&TargetInfo{username: username, password: password})
		}
	}

	wg.Wait()

	return nil
}

func (runner *Runner) start(username, password string, m any) error {
	service := runner.options.Service
	if service == SSH_NAME {
		sshclient := m.(*ssh.SSH)
		return sshclient.AuthSSHRtries(username, password)
	}
	return nil
}

func (runner *Runner) validateService() (any, error) {
	service := runner.options.Service
	if service == SSH_NAME {
		m, err := ssh.NewSSH(runner.options.Host, runner.options.Port, runner.options.Retries)
		if err != nil {
			return m, err
		}
		return m, nil
	}

	return nil, errors.New("error")
}
