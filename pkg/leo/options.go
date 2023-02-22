package leo

import (
	"errors"
	"net"
	"strings"

	"github.com/zan8in/goflags"
	"github.com/zan8in/gologger"
	"github.com/zan8in/gologger/levels"
	"github.com/zan8in/leo/pkg/utils/dateutil"
	"github.com/zan8in/leo/pkg/utils/fileutil"
	"github.com/zan8in/leo/pkg/utils/iputil"
)

type Options struct {
	// target 'protocol://host:port' to crack
	Target string
	// host to crack logon
	Host string
	// path to file containing a list of target hosts to crack(one per line)
	HostFile string
	// ports to crack
	Port string
	// crack supports the protocols: ssh,ftp,mssql
	Service string
	// login with LOGIN name for (comma-separated)
	User string
	// try password PASS for (comma-separated)
	Password string
	// load several logins from FILE
	UserFile string
	// load several passwords from FILE
	PasswordFile string
	// disable progress bar
	Silent bool
	// update Leo engine to the latest released version
	Update bool
	// time to wait in seconds before timeout
	Timeout int
	// number of times to retry a failed request
	Retries int
	// show all crack log
	Debug bool
	// maximum number of requests to send per second (150)
	RateLimit int
	// maximum number of crack to be executed in parallel (25)
	Concurrency int
	// write found login/password pairs to FILE
	Output string

	// Crack credential host information including host, port number and protocol
	Hosts []HostInfo
	// Crack login account credentials
	Users []string
	// Cracking Login Password Credentials
	Passwords []string

	// Crack the maximum total number of requests
	Count uint32
	// Crack the current number of requests
	CurrentCount uint32

	// List of successfully cracked results
	SuccessList []string
	// List of hosts that failed to crack
	FailedMap map[string]int
}

type HostInfo struct {
	Host    string
	Port    string
	Service string
}

func ParseOptions() *Options {

	ShowBanner()

	options := &Options{
		Count:        0,
		CurrentCount: 0,
		SuccessList:  []string{},
		FailedMap:    map[string]int{},
	}

	flagSet := goflags.NewFlagSet()
	flagSet.SetDescription(`Leo`)

	flagSet.CreateGroup("target", "target",
		flagSet.StringVarP(&options.Target, "t", "", "", "target 'protocol://host:port' to crack"),
		flagSet.StringVarP(&options.Host, "h", "", "", "host to crack logon"),
		flagSet.StringVarP(&options.HostFile, "H", "", "", "path to file containing a list of target hosts to crack(one per line)"),
		flagSet.StringVarP(&options.Service, "s", "", "", "crack supports the protocols: ssh,ftp,mssql"),
		flagSet.StringVarP(&options.Port, "port", "", "", "ports to crack"),
	)

	flagSet.CreateGroup("credentials", "credentials",
		flagSet.StringVarP(&options.User, "l", "", "", "login with LOGIN name for (comma-separated)"),
		flagSet.StringVarP(&options.Password, "p", "", "", "try password PASS for (comma-separated)"),
		flagSet.StringVarP(&options.UserFile, "L", "", "", "load several logins from FILE"),
		flagSet.StringVarP(&options.PasswordFile, "P", "", "", "load several passwords from FILE"),
	)

	flagSet.CreateGroup("rate-limit", "Rate-Limit",
		flagSet.IntVarP(&options.RateLimit, "rate-limit", "rl", 150, "maximum number of requests to send per second"),
		flagSet.IntVarP(&options.Concurrency, "concurrency", "c", 25, "maximum number of crack to be executed in parallel"),
	)

	flagSet.CreateGroup("optimization", "Optimizations",
		flagSet.IntVar(&options.Retries, "retries", 2, "number of times to retry a failed request"),
		flagSet.IntVar(&options.Timeout, "timeout", 10, "time to wait in seconds before timeout"),
		flagSet.BoolVar(&options.Silent, "silent", false, "disable progress bar"),
		flagSet.StringVarP(&options.Output, "o", "", "", "write found login/password pairs to FILE"),
	)

	flagSet.CreateGroup("update", "Update",
		flagSet.BoolVar(&options.Update, "update", false, "update Leo engine to the latest released version"),
	)

	flagSet.CreateGroup("debug", "Debug",
		flagSet.BoolVar(&options.Debug, "debug", false, "show all crack log"),
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
		hostlist, err := fileutil.ReadFileLineByLine(options.HostFile)
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
				options.FailedMap[host] = 1
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
		userlist, err := fileutil.ReadFileLineByLine(options.UserFile)
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
		passlist, err := fileutil.ReadFileLineByLine(options.PasswordFile)
		if err != nil {
			return
		}
		options.Passwords = append(options.Passwords, passlist...)
	}
}

func (options *Options) showBanner() {
	gologger.Print().Msgf("Cracking login credentials on [%s] protocol network. Start time %s",
		options.Service,
		dateutil.GetNowFullDateTime(),
	)
}
