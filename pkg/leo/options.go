package leo

import (
	"errors"
	"net"
	"strings"

	"github.com/zan8in/goflags"
	"github.com/zan8in/gologger"
	"github.com/zan8in/gologger/levels"
	"github.com/zan8in/leo/pkg/utils"
	"github.com/zan8in/leo/pkg/utils/iputil"
)

type Options struct {
	// -t ssh://192.168.88.168:22
	Target string

	// -h 127.0.0.1 or 127.0.0.1,192.168.1.1,192.168.1.2
	Host string
	// -H hosts.txt
	HostFile string

	// -port
	Port string

	// -s  ssh
	Service string

	// -l root or root,admin,test
	User string
	// -p 123456  or  123456,123,111
	Password string

	// -L username.txt
	UserFile string
	// -P password.txt
	PasswordFile string

	// no progress if silent is true
	Silent bool

	// no progress if silent is true
	Update bool

	// -t Timeout
	Timeout int

	// number of times to retry a failed request (default 1)
	Retries int

	// -d DEBUG
	Debug bool

	// maximum number of requests to send per second (default 150)
	RateLimit int

	// maximum number of afrog-pocs to be executed in parallel (default 25)
	Concurrency int

	// write found login/password pairs to FILE instead of stdout
	Output string

	Hosts     []HostInfo
	Users     []string
	Passwords []string

	Count        uint32
	CurrentCount uint32

	SuccessList []string
}

type HostInfo struct {
	Host    string
	Port    string
	Service string
}

func ParseOptions() *Options {

	ShowBanner()

	options := &Options{Count: 0, CurrentCount: 0, SuccessList: []string{}}

	flagSet := goflags.NewFlagSet()
	flagSet.SetDescription(`Leo`)

	flagSet.CreateGroup("target", "target",
		flagSet.StringVarP(&options.Target, "t", "", "", "-t ssh://192.168.66.100:22"),
		flagSet.StringVarP(&options.Host, "h", "", "", "-h 192.168.66.100"),
		flagSet.StringVarP(&options.HostFile, "H", "", "", "-H hostlist.txt"),
		flagSet.StringVarP(&options.Service, "s", "", "", "supports the protocols: ssh,ftp,mssql"),
		flagSet.StringVarP(&options.Port, "port", "", "", "-port 22"),
	)

	flagSet.CreateGroup("credentials", "credentials",
		flagSet.StringVarP(&options.User, "l", "", "", "login with LOGIN name"),
		flagSet.StringVarP(&options.Password, "p", "", "", "try password PASS"),
		flagSet.StringVarP(&options.UserFile, "L", "", "", "load several logins from FILE"),
		flagSet.StringVarP(&options.PasswordFile, "P", "", "", "load several passwords from FILE"),
	)

	flagSet.CreateGroup("rate-limit", "Rate-Limit",
		flagSet.IntVarP(&options.RateLimit, "rate-limit", "rl", 150, "maximum number of requests to send per second"),
		flagSet.IntVarP(&options.Concurrency, "concurrency", "c", 25, "maximum number of afrog-pocs to be executed in parallel"),
	)

	flagSet.CreateGroup("optimization", "Optimizations",
		flagSet.IntVar(&options.Retries, "retries", 10, "number of times to retry a failed request"),
		flagSet.IntVar(&options.Timeout, "timeout", 10, "time to wait in seconds before timeout"),
		flagSet.BoolVar(&options.Silent, "silent", false, "no progress, only results"),
		flagSet.StringVarP(&options.Output, "o", "", "", "write found login/password pairs to FILE"),
	)

	flagSet.CreateGroup("update", "Update",
		flagSet.BoolVar(&options.Update, "update", false, "update leo engine to the latest released version"),
	)

	flagSet.CreateGroup("debug", "Debug",
		flagSet.BoolVar(&options.Debug, "debug", false, ""),
	)

	_ = flagSet.Parse()

	err := options.validateOptions()
	if err != nil {
		gologger.Fatal().Msgf("Program exiting: %s\n", err)
	}

	return options
}

func (options *Options) validateOptions() error {
	if len(options.Target) == 0 && len(options.Host) == 0 && len(options.HostFile) == 0 {
		return ErrNoTargetOrHost

	} else if len(options.Target) > 0 {
		targetService := strings.Split(options.Target, "://")
		if len(targetService) != 2 {
			return ErrTargetFormat
		}

		options.Service = strings.ToLower(strings.TrimSpace(targetService[0]))

		defaultPort := DefaultServicePort[options.Service]
		if len(defaultPort.Port) == 0 {
			return ErrNoService
		}

		if len(options.Port) == 0 {
			options.Port = defaultPort.Port
		}

		if err := options.handleHost(targetService[1]); err != nil {
			return err
		}

	} else if len(options.Service) == 0 {
		return ErrNoService

	} else if len(options.Host) > 0 {
		options.Service = strings.ToLower(strings.TrimSpace(options.Service))

		defaultPort := DefaultServicePort[options.Service]
		if len(defaultPort.Port) == 0 {
			return ErrNoService
		}

		if len(options.Port) == 0 {
			options.Port = defaultPort.Port
		}

		if err := options.handleHost(options.Host); err != nil {
			return err
		}

	} else if len(options.HostFile) > 0 {
		hostlist, err := utils.ReadFileLineByLine(options.HostFile)
		if err != nil {
			return ErrNoTargetOrHost
		}

		defaultPort := DefaultServicePort[options.Service]
		if len(defaultPort.Port) == 0 {
			return ErrNoService
		}

		if len(options.Port) == 0 {
			options.Port = defaultPort.Port
		}

		if len(hostlist) == 0 {
			return ErrNoTargetOrHost
		}

		for _, host := range hostlist {
			if options.handleHost(host) != nil {
				continue
			}
		}

	} else {
		return ErrNoOther
	}

	if options.Debug {
		gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
	}

	return nil
}

func (options *Options) handleHost(host string) error {
	splitHost := strings.Split(host, ":")
	if len(splitHost) > 2 {
		return errors.New(host + " format error")
	}

	if len(splitHost) == 1 {
		options.Hosts = append(options.Hosts, HostInfo{Host: host, Port: options.Port, Service: options.Service})
	}

	if len(splitHost) == 2 {
		ip, port, err := net.SplitHostPort(host)
		if err != nil {
			return err
		}
		isPort := iputil.IsPort(port)
		if !isPort {
			return errors.New(host + " format error, " + splitHost[1] + " is not port")
		}
		options.Hosts = append(options.Hosts, HostInfo{Host: ip, Port: port, Service: options.Service})
	}

	return nil
}

func (options *Options) convertUsers() {
	if len(options.User) > 0 {
		if !strings.Contains(options.User, ",") {
			options.Users = append(options.Users, options.User)
		} else {
			options.Users = append(options.Users, strings.Split(options.User, ",")...)
		}
	}
	if len(options.UserFile) > 0 {
		userlist, err := utils.ReadFileLineByLine(options.UserFile)
		if err != nil {
			return
		}
		options.Users = append(options.Users, userlist...)
	}
}

func (options *Options) convertPasswords() {
	if len(options.Password) > 0 {
		if !strings.Contains(options.Password, ",") {
			options.Passwords = append(options.Passwords, options.Password)
		} else {
			options.Passwords = append(options.Passwords, strings.Split(options.Password, ",")...)
		}
	}
	if len(options.PasswordFile) > 0 {
		passlist, err := utils.ReadFileLineByLine(options.PasswordFile)
		if err != nil {
			return
		}
		options.Passwords = append(options.Passwords, passlist...)
	}
}

func (options *Options) showBanner() {
	if len(options.Target) > 0 {
		gologger.Info().Msgf("Target: %s", strings.ToLower(options.Target))
	}

	if len(options.Service) > 0 {
		gologger.Info().Msgf("Service: %s", options.Service)
	}

	if len(options.Host) > 0 {
		gologger.Info().Msgf("Host: %s", options.Host)
	}

	if len(options.HostFile) > 0 {
		gologger.Info().Msgf("Host File: %s", options.HostFile)
	}

	if len(options.Port) > 0 {
		gologger.Info().Msgf("Port: %s", options.Port)
	}

	if len(options.User) > 0 {
		gologger.Info().Msgf("User: %s", options.User)
	}

	if len(options.UserFile) > 0 {
		gologger.Info().Msgf("User File: %s", options.UserFile)
	}

	if len(options.Password) > 0 {
		gologger.Info().Msgf("Password: %s", options.Password)
	}

	if len(options.PasswordFile) > 0 {
		gologger.Info().Msgf("Password File: %s", options.PasswordFile)
	}
}
