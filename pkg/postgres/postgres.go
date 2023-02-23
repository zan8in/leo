package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type POSTGRES struct {
	host    string
	port    string
	retries int
	timeout int
}

var (
	ErrRtries             = errors.New("retries exceeded")
	ErrNoHost             = errors.New("no input host provided")
	default_postgres_port = "5432"

	ErrLoginFailed = "Login failed for user"
)

func New(host, port string, retries, timeout int) (*POSTGRES, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_postgres_port
	}

	postgres := &POSTGRES{host: host, port: port, retries: retries, timeout: timeout}

	return postgres, nil
}

func (postgres *POSTGRES) AuthRetries(user, password string) (err error) {
	sum := 0
	for {
		if sum > postgres.retries {
			return errors.New(ErrRtries.Error() + ", " + err.Error())
		}

		err = postgres.auth(user, password)
		if err != nil && strings.Contains(err.Error(), ErrLoginFailed) {
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

func (postgres *POSTGRES) auth(user, password string) error {
	connString := fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=%v", user, password, postgres.host, postgres.port, "postgres", "disable")
	conn, err := sql.Open("postgres", connString)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.Ping()

	return err
}
