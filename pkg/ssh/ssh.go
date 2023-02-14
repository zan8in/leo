package ssh

import (
	"errors"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSH struct {
	host   string
	port   string
	config ssh.ClientConfig
	rtries int
}

var (
	ErrNoSSH         = errors.New("no live ssh")
	ErrRtries        = errors.New("retries exceeded")
	ErrNoHost        = errors.New("no input host provided")
	default_ssh_port = "22"
	keyExchanges     = []string{"diffie-hellman-group-exchange-sha256", "diffie-hellman-group14-sha256", "diffie-hellman-group1-sha1", "diffie-hellman-group14-sha1"}
)

func NewSSH(host, port string, rtries int) (*SSH, error) {
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

	ssh := &SSH{host: host, port: port, config: config, rtries: rtries}

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
		// if err != nil && strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host") && sum <= s.rtries {
		// 	sum++
		// 	continue
		// } else if err != nil && strings.Contains(err.Error(), "handshake failed: EOF") && sum <= s.rtries {
		// 	sum++
		// 	continue
		// } else if err != nil && strings.Contains(err.Error(), "An established connection was aborted by the software in your host machine") && sum <= s.rtries {
		// 	sum++
		// 	continue
		// } else if err != nil {
		// 	return err
		// }
		if err != nil && !strings.Contains(err.Error(), "unable to authenticate") {
			sum++
			time.Sleep(500 * time.Millisecond)
			continue
		} else if err != nil {
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

	return nil
}

func (s *SSH) checkRtries() error {
	sum := 0
	for {
		if sum > s.rtries {
			return ErrRtries
		}

		_, err := s.check()
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
	s.config.User = "username"
	s.config.Auth = []ssh.AuthMethod{ssh.Password("password")}

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
