package oracle

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

type ORACLE struct {
	host    string
	port    string
	retries int
	timeout int
}

var (
	ErrRtries           = errors.New("retries exceeded")
	ErrNoHost           = errors.New("no input host provided")
	default_oracle_port = "1521"

	ErrLoginFailed = "Login failed for user"
)

func New(host, port string, retries, timeout int) (*ORACLE, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_oracle_port
	}

	oracle := &ORACLE{host: host, port: port, retries: retries, timeout: timeout}

	return oracle, nil
}

func (oracle *ORACLE) AuthRetries(user, password string) (err error) {
	sum := 0
	for {
		if sum > oracle.retries {
			return errors.New(ErrRtries.Error() + ", " + err.Error())
		}

		err = oracle.auth(user, password)
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

func (oracle *ORACLE) auth(user, password string) error {
	connString := fmt.Sprintf("oracle://%s:%s@%s:%s/orcl", user, password, oracle.host, oracle.port)
	conn, err := sql.Open("oracle", connString)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.Ping()

	return err
}
