package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MYSQL struct {
	host            string
	port            string
	retries         int
	checkLiveRtries int
}

var (
	ErrNoMYSQL         = errors.New("mysql connection failed")
	ErrRtries          = errors.New("retries exceeded")
	ErrNoHost          = errors.New("no input host provided")
	ErrNoSession       = errors.New("no session")
	default_mysql_port = "3306"
	check_live_rtries  = 3

	TlsErr = "TLS requested but server does not support TLS"
)

func NewMYSQL(host, port string, retries, Timeout int) (*MYSQL, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_mysql_port
	}

	mysql := &MYSQL{host: host, port: port, retries: retries, checkLiveRtries: check_live_rtries}

	return mysql, nil
}

func (mysql *MYSQL) AuthMYSQLRtries(user, password string) (err error) {
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
