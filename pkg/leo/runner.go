package leo

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/remeh/sizedwaitgroup"
	"github.com/zan8in/gologger"
	"github.com/zan8in/leo/pkg/utils"
)

var Ticker *time.Ticker

type Runner struct {
	options      *Options
	execute      *Execute
	callbackchan chan *CallbackInfo

	syncfile *utils.Syncfile
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
	var err error

	runner := &Runner{
		options:      options,
		callbackchan: make(chan *CallbackInfo),
		syncfile:     &utils.Syncfile{},
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

	if len(options.Output) > 0 {
		runner.syncfile, err = utils.NewSyncfile(options.Output)
		if err != nil {
			return runner, err
		}
	}

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
	if strings.Contains(password, "%upper-user%") {
		return strings.ReplaceAll(password, "%upper-user%", strings.ToUpper(username)[:1]+username[1:])
	}
	return password
}

func (runner *Runner) Listener() {

	defer close(runner.callbackchan)

	starttime := time.Now()

	for result := range runner.callbackchan {
		port, service, host, user, pass := runner.options.Port, runner.options.Service, result.Host, result.Username, result.Password

		if result.Err == nil && result.Status != STATUS_COMPLATE {
			info := fmt.Sprintf("\r[%s][%s] %s %s %s\r\n", port, service, host, user, pass)

			gologger.Print().Msgf(info)

			if len(runner.options.Output) > 0 {
				go func() {
					runner.syncfile.Write(strings.TrimSpace(info) + "\n")
					runner.options.SuccessList = append(runner.options.SuccessList, strings.TrimSpace(info))
				}()
			}
		}
		if result.Err == nil && result.Status == STATUS_COMPLATE {
			time.Sleep(3 * time.Second)
			break
		}
		if result.Err != nil && result.Status != STATUS_FAILED {
			gologger.Debug().Msgf("\r[%s][%s] %s %s %s, %s\r\n", port, service, host, user, pass, result.Err.Error())
		}
		if result.Err != nil && result.Status == STATUS_FAILED {
			gologger.Error().Msgf("\r[%s][%s] %s, Connection failed, %s\r\n", port, service, host, result.Err.Error())
		}
		if !runner.options.Silent {
			fmt.Printf("\r%d/%d/%d%%/%s", result.CurrentCount, runner.options.Count, result.CurrentCount*100/runner.options.Count,
				strings.Split(time.Since(starttime).String(), ".")[0]+"s")
		}
	}

	gologger.Print().Msgf("")
	gologger.Print().Msgf("%d of %d target successfully completed, %d valid password found\r\n",
		len(runner.options.Hosts), len(runner.options.Hosts), len(runner.options.SuccessList))

	if len(runner.options.SuccessList) > 0 && len(runner.options.Output) > 0 {
		// for _, info := range runner.options.SuccessList {
		// 	err := utils.WriteString(runner.options.Output, info+"\r\n")
		// 	if err != nil {
		// 		gologger.Fatal().Msgf("write file failed, %s\r\n", err.Error())
		// 	}
		// }
		gologger.Print().Msgf("write found login/password pairs to FILE: %s", runner.options.Output)
	}

	gologger.Print().Msgf("Leo finished at %s\r\n", utils.GetNowDateTime())

}
