package ftp

import (
	"errors"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

type FTP struct {
	host    string
	port    string
	retries int
	timeout int
}

var (
	ErrRtries        = errors.New("retries exceeded")
	ErrNoHost        = errors.New("no input host provided")
	default_ftp_port = "21"

	ErrLogin   = "Login incorrect"
	Err500OOPS = "cannot change directory"
)

func New(host, port string, retries, timeout int) (*FTP, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_ftp_port
	}

	ftp := &FTP{host: host, port: port, retries: retries, timeout: timeout}

	return ftp, nil
}

func (f *FTP) AuthRetries(user, password string) (err error) {
	sum := 0
	for {
		if sum > f.retries {
			return errors.New(ErrRtries.Error() + ", " + err.Error())
		}

		err = f.auth(user, password)
		if err != nil {
			if strings.Contains(err.Error(), Err500OOPS) {
				return nil
			}
			if !strings.Contains(err.Error(), ErrLogin) {
				sum++
				time.Sleep(500 * time.Millisecond)
				continue
			}
		}

		return err
	}
}

func (f *FTP) auth(user, password string) (err error) {
	conn, err := ftp.Dial(f.host+":"+f.port, ftp.DialWithTimeout(time.Duration(f.timeout)*time.Second))
	if err != nil {
		return err
	}

	err = conn.Login(user, password)
	if err != nil {
		return err
	}

	_, err = conn.CurrentDir()
	if err != nil {
		return err
	}

	return err
}
