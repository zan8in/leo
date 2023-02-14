package leo

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants"
	"github.com/remeh/sizedwaitgroup"
	"github.com/zan8in/gologger"
	"github.com/zan8in/leo/pkg/ssh"
)

var Ticker *time.Ticker

type Runner struct {
	options *Options
}

type TargetInfo struct {
	host     string
	username string
	password string
	m        any
}

type CallbackInfo struct {
	Host         string
	Username     string
	Password     string
	Err          error
	CurrentCount uint32
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

	options.Count = uint32(len(options.Hosts) * len(options.Users) * len(options.Passwords))

	options.showBanner()

	gologger.Print().Msg("")

	return runner, nil
}

func (runner *Runner) Run(acb ApiCallBack) {

	runner.options.ApiCallBack = acb

	Ticker = time.NewTicker(time.Second / time.Duration(runner.options.RateLimit))
	var wg sync.WaitGroup

	p, _ := ants.NewPoolWithFunc(runner.options.Concurrency, func(p any) {
		defer wg.Done()
		<-Ticker.C

		ti := p.(*TargetInfo)

		err := runner.start(ti.host, ti.username, ti.password, ti.m)

		atomic.AddUint32(&runner.options.CurrentCount, 1)
		runner.options.ApiCallBack(&CallbackInfo{Err: err, Host: ti.host, Username: ti.username, Password: ti.password, CurrentCount: runner.options.CurrentCount})
	})
	defer p.Release()

	swg := sizedwaitgroup.New(runtime.NumCPU())
	for _, host := range runner.options.Hosts {
		m, err := runner.validateService(host)
		if err != nil {

			atomic.AddUint32(&runner.options.CurrentCount, uint32(len(runner.options.Users)*len(runner.options.Passwords)))
			runner.options.ApiCallBack(&CallbackInfo{Err: err, Host: host, Username: "", Password: "", CurrentCount: runner.options.CurrentCount})

			continue
		}

		swg.Add()
		go func(host string, m any) {
			defer swg.Done()
			for _, username := range runner.options.Users {
				for _, password := range runner.options.Passwords {
					wg.Add(1)
					p.Invoke(&TargetInfo{host: host, username: username, password: handlePassword(username, password), m: m})
				}
			}
		}(host, m)
	}
	swg.Wait()

	wg.Wait()
}

func (runner *Runner) start(host, username, password string, m any) error {
	service := runner.options.Service
	if service == SSH_NAME {
		sshclient := m.(*ssh.SSH)
		return sshclient.AuthSSHRtries(host, username, password)
	}
	return nil
}

func (runner *Runner) validateService(host string) (any, error) {
	service := runner.options.Service
	if service == SSH_NAME {
		m, err := ssh.NewSSH(host, runner.options.Port, runner.options.Retries)
		if err != nil {
			return m, err
		}
		return m, nil
	}

	return nil, errors.New("error")
}

func handlePassword(username, password string) string {
	if strings.Contains(password, "%user%") {
		return strings.ReplaceAll(password, "%user%", username)
	}
	return password
}
