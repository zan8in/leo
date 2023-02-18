package leo

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants"
	"github.com/remeh/sizedwaitgroup"
	"github.com/zan8in/gologger"
)

var Ticker *time.Ticker

type Runner struct {
	options      *Options
	execute      *Execute
	callbackchan chan *CallbackInfo
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
	Status       int
}

func NewRunner(options *Options) (*Runner, error) {
	runner := &Runner{
		options:      options,
		callbackchan: make(chan *CallbackInfo),
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

	options.showBanner()

	gologger.Print().Msg("")

	runner.execute = NewExecute(options)

	return runner, nil
}

func (runner *Runner) Run() {
	go func() {
		Ticker = time.NewTicker(time.Second / time.Duration(runner.options.RateLimit))
		var wg sync.WaitGroup

		p, _ := ants.NewPoolWithFunc(runner.options.Concurrency, func(p any) {
			defer wg.Done()
			<-Ticker.C

			ti := p.(*TargetInfo)

			err := runner.execute.start(ti.host, ti.username, ti.password, ti.m)

			atomic.AddUint32(&runner.options.CurrentCount, 1)
			runner.callbackchan <- &CallbackInfo{Err: err, Host: ti.host, Username: ti.username, Password: ti.password, CurrentCount: runner.options.CurrentCount}
		})
		defer p.Release()

		swg := sizedwaitgroup.New(runtime.NumCPU())
		for _, host := range runner.options.Hosts {

			swg.Add()
			go func(host string) {
				defer swg.Done()

				m, err := runner.execute.validateService(host)
				if err != nil {

					atomic.AddUint32(&runner.options.CurrentCount, uint32(len(runner.options.Users)*len(runner.options.Passwords)))
					runner.callbackchan <- &CallbackInfo{Err: err, Host: host, Username: "", Password: "", CurrentCount: runner.options.CurrentCount, Status: STATUS_FAILED}

					return
				}

				for _, username := range runner.options.Users {
					for _, password := range runner.options.Passwords {
						wg.Add(1)
						p.Invoke(&TargetInfo{host: host, username: username, password: handlePassword(username, password), m: m})
					}
				}
			}(host)
		}
		swg.Wait()

		wg.Wait()

		runner.callbackchan <- &CallbackInfo{Err: nil, Host: "", Username: "", Password: "", CurrentCount: runner.options.CurrentCount, Status: STATUS_COMPLATE}
	}()
}

func handlePassword(username, password string) string {
	if strings.Contains(password, "%user%") {
		return strings.ReplaceAll(password, "%user%", username)
	}
	return password
}

func (runner *Runner) Listener() {

	defer close(runner.callbackchan)

	starttime := time.Now()

	for result := range runner.callbackchan {
		if result.Err == nil && result.Status != STATUS_COMPLATE {
			gologger.Print().Msgf("\r[%s][%s][%s] username: %s password: %s |||||||||||||||||||||||||||||||||\r\n", runner.options.Service, result.Host, runner.options.Port, result.Username, result.Password)
		}
		if result.Err == nil && result.Status == STATUS_COMPLATE {
			return
		}
		if result.Err != nil && result.Status != STATUS_FAILED {
			gologger.Debug().Msgf("\r[%s][%s][%s] username: %s password: %s, %s\r\n", runner.options.Service, result.Host, runner.options.Port, result.Username, result.Password, result.Err.Error())
		}
		if result.Err != nil && result.Status == STATUS_FAILED {
			gologger.Error().Msgf("[%s][%s][%s] Connection failed, %s\r\n", runner.options.Service, result.Host, runner.options.Port, result.Err.Error())
		}
		if !runner.options.Silent {
			fmt.Printf("\r%d/%d/%d%%/%s", result.CurrentCount, runner.options.Count, result.CurrentCount*100/runner.options.Count, strings.Split(time.Since(starttime).String(), ".")[0]+"s")
		}
	}

}
