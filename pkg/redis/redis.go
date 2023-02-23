package redis

import (
	"errors"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

type REDIS struct {
	host    string
	port    string
	retries int
	timeout int
}

var (
	ErrRtries          = errors.New("retries exceeded")
	ErrNoHost          = errors.New("no input host provided")
	default_redis_port = "6379"

	ErrInvalidPass = "ERR invalid password"
	ErrWrongPass   = "WRONGPASS invalid username-password pair"
)

func New(host, port string, retries, timeout int) (*REDIS, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_redis_port
	}

	timeout = 3
	redis := &REDIS{host: host, port: port, retries: retries, timeout: timeout}

	return redis, nil
}

func (redis *REDIS) AuthRetries(user, password string) (err error) {
	sum := 0
	for {
		if sum > redis.retries {
			return errors.New(ErrRtries.Error() + ", " + err.Error())
		}

		err = redis.auth(user, password)
		if err != nil && (strings.Contains(err.Error(), ErrInvalidPass) || strings.Contains(err.Error(), ErrWrongPass)) {
			return err
		}
		if err != nil {
			sum++
			time.Sleep(500 * time.Millisecond)
			continue
		}

		return nil
	}
}

func (r *REDIS) auth(user, password string) error {
	conn, err := redis.Dial("tcp", r.host+":"+r.port,
		redis.DialPassword(password),
		redis.DialConnectTimeout(time.Duration(r.timeout)*time.Second),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	return err
}
