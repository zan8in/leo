package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MYSQL struct {
	host    string
	port    string
	retries int
}

var (
	ErrRtries          = errors.New("retries exceeded")
	ErrNoHost          = errors.New("no input host provided")
	default_mysql_port = "3306"

	ErrBusyBuffer  = "busy buffer"
	ErrOldPassword = "this user requires old password authentication"
)

func New(host, port string, retries, Timeout int) (*MYSQL, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_mysql_port
	}

	mysql := &MYSQL{host: host, port: port, retries: retries}

	return mysql, nil
}

func (mysql *MYSQL) AuthRetries(user, password string) (err error) {
	sum := 0
	for {
		if sum > mysql.retries {
			return errors.New(ErrRtries.Error() + ", " + err.Error())
		}

		err = mysql.auth(user, password)
		if err != nil {
			sum++
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err != nil && strings.Contains(err.Error(), ErrOldPassword) {
			return nil
		}

		return nil
	}
}

func (mysql *MYSQL) auth(user, password string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql?charset=utf8", user, password, mysql.host, mysql.port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	err = db.Ping()

	db.Close()

	return err
}
