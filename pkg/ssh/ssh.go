package ssh

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSH struct {
	host            string
	port            string
	config          ssh.ClientConfig
	rtries          int
	checkLiveRtries int
}

var (
	ErrNoSSH          = errors.New("ssh connection failed")
	ErrRtries         = errors.New("retries exceeded")
	ErrNoHost         = errors.New("no input host provided")
	default_ssh_port  = "22"
	check_live_rtries = 3
	keyExchanges      = []string{"diffie-hellman-group-exchange-sha256", "diffie-hellman-group14-sha256", "diffie-hellman-group1-sha1", "diffie-hellman-group14-sha1"}
)

func NewSSH(host, port string, rtries, Timeout int) (*SSH, error) {
	if len(host) == 0 {
		return nil, ErrNoHost
	}

	if len(port) == 0 {
		port = default_ssh_port
	}

	config := ssh.ClientConfig{
		// HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	config.KeyExchanges = append(config.KeyExchanges, keyExchanges...)

	ssh := &SSH{host: host, port: port, config: config, rtries: rtries, checkLiveRtries: check_live_rtries}

	err := ssh.checkRtries()
	if err != nil {
		return ssh, err
	}

	return ssh, nil
}

func (s *SSH) AuthSSHRtries(host, username, password string) error {
	sum := 0
	for {
		if sum > s.rtries {
			return ErrRtries
		}

		err := s.authSSH(host, username, password)
		if err != nil && !strings.Contains(err.Error(), "unable to authenticate") {
			sum++
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err != nil {
			return err
		}

		return nil
	}
}

func (s *SSH) authSSH(host, username, password string) error {
	s.host = host
	s.config.User = username
	s.config.Auth = []ssh.AuthMethod{ssh.Password(password)}

	client, err := ssh.Dial("tcp", s.host+":"+s.port, &s.config)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("ls"); err != nil {
		return err
	}

	if b.Len() > 0 {
		return nil
	}

	return err
}

func (s *SSH) checkRtries() error {
	var err error
	sum := 0
	for {
		if sum > s.checkLiveRtries {
			return errors.New(ErrRtries.Error() + ", " + err.Error())
		}

		_, err = s.check()
		if err != nil && !strings.Contains(err.Error(), "No connection could be made") {
			sum++
			time.Sleep(500 * time.Millisecond)
			continue
		} else if err != nil {
			return err
		}

		return nil
	}
}

func (s *SSH) check() (bool, error) {
	s.config.User = "root"
	s.config.Auth = []ssh.AuthMethod{ssh.Password("root")}

	client, err := ssh.Dial("tcp", s.host+":"+s.port, &s.config)
	if err != nil {
		if strings.Contains(err.Error(), "unable to authenticate") {
			return true, nil
		}
		return false, err
	}
	defer client.Close()

	return true, nil
}
