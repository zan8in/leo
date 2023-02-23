package mssql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

type MSSQL struct {
	host    string
	port    string
	retries int
	timeout int
}

var (
	ErrRtries          = errors.New("retries exceeded")
	ErrNoHost          = errors.New("no input host provided")
	default_mssql_port = "3306"

	ErrLoginFailed = "Login failed for user"
)

func New(host, port string, retries, timeout int) (*MSSQL, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_mssql_port
	}

	mssql := &MSSQL{host: host, port: port, retries: retries, timeout: timeout}

	return mssql, nil
}

func (mssql *MSSQL) AuthRetries(user, password string) (err error) {
	sum := 0
	for {
		if sum > mssql.retries {
			return errors.New(ErrRtries.Error() + ", " + err.Error())
		}

		err = mssql.auth(user, password)
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

func (mssql *MSSQL) auth(user, password string) error {
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;encrypt=disable;timeout=%v",
		mssql.host, user, password, mssql.port, time.Duration(mssql.timeout))
	conn, err := sql.Open("mssql", connString)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.Ping()

	return err
}
