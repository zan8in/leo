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
	"github.com/zan8in/leo/pkg/utils/dateutil"
	"github.com/zan8in/leo/pkg/utils/fileutil"
)

var Ticker *time.Ticker

type Runner struct {
	options      *Options
	execute      *Execute
	callbackchan chan *CallbackInfo

	syncfile *fileutil.Syncfile
}

type HostCredentials struct {
	Host string
	Port string
	User string
	Pass string
	M    any
}

type CallbackInfo struct {
	HostInfo        HostInfo
	HostCredentials HostCredentials
	Err             error
	CurrentCount    uint32
	Status          int
}

func NewRunner(options *Options) (*Runner, error) {
	var err error

	runner := &Runner{
		options:      options,
		callbackchan: make(chan *CallbackInfo),
		syncfile:     &fileutil.Syncfile{},
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
		runner.syncfile, err = fileutil.NewSyncfile(options.Output)
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

			hc := p.(*HostCredentials)

			err := runner.execute.start(hc.Host, hc.User, hc.Pass, hc.M)

			atomic.AddUint32(&runner.options.CurrentCount, 1)
			runner.callbackchan <- &CallbackInfo{
				Err:             err,
				HostInfo:        HostInfo{Host: hc.Host, Port: hc.Port},
				HostCredentials: *hc,
				CurrentCount:    runner.options.CurrentCount,
			}
		})
		defer p.Release()

		swg := sizedwaitgroup.New(runtime.NumCPU())
		for _, host := range runner.options.Hosts {

			swg.Add()
			go func(host HostInfo) {
				defer swg.Done()

				m, err := runner.execute.validateService(host.Host, host.Port)
				if err != nil {

					atomic.AddUint32(&runner.options.CurrentCount, uint32(len(runner.options.Users)*len(runner.options.Passwords)))
					runner.callbackchan <- &CallbackInfo{
						Err:             err,
						HostInfo:        host,
						HostCredentials: HostCredentials{},
						CurrentCount:    runner.options.CurrentCount,
						Status:          STATUS_FAILED,
					}

					return
				}

				for _, username := range runner.options.Users {
					for _, password := range runner.options.Passwords {
						wg.Add(1)
						p.Invoke(&HostCredentials{
							Host: host.Host,
							Port: host.Port,
							User: username,
							Pass: handlePassword(username, password),
							M:    m,
						})
					}
				}
			}(host)
		}
		swg.Wait()

		wg.Wait()

		runner.callbackchan <- &CallbackInfo{
			Err:             nil,
			HostInfo:        HostInfo{},
			HostCredentials: HostCredentials{},
			CurrentCount:    runner.options.CurrentCount,
			Status:          STATUS_COMPLATE,
		}

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
		port, service, host := result.HostInfo.Port, runner.options.Service, result.HostInfo.Host
		user, pass := result.HostCredentials.User, result.HostCredentials.Pass

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
			fmt.Printf("\r%d/%d/%d%%/%s",
				result.CurrentCount,
				runner.options.Count,
				result.CurrentCount*100/runner.options.Count,
				strings.Split(time.Since(starttime).String(), ".")[0]+"s",
			)
		}
	}

	gologger.Print().Msgf("")
	gologger.Print().Msgf("%d of %d target successfully completed, %d valid password found\r\n",
		len(runner.options.Hosts), len(runner.options.Hosts), len(runner.options.SuccessList))

	if len(runner.options.SuccessList) > 0 && len(runner.options.Output) > 0 {
		gologger.Print().Msgf("write found login/password pairs to FILE: %s", runner.options.Output)
	}

	gologger.Print().Msgf("Leo finished at %s\r\n", dateutil.GetNowDateTime())

}
