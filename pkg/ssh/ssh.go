package ssh

import (
	"errors"
	"fmt"
	"net"
	"strings"

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
	default_ssh_host = "0.0.0.0"
	default_ssh_port = "22"
)

func NewSSH(host, port string, rtries int) (*SSH, error) {
	if len(host) == 0 {
		host = default_ssh_host
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

	ssh := &SSH{host: host, port: port, config: config, rtries: rtries}

	flag, err := ssh.check()
	if !flag {
		return ssh, errors.New(fmt.Sprintf("%s"+err.Error(), ErrNoSSH))
	}

	return ssh, nil
}

func SSH_TEST(host, port string) error {
	if len(host) == 0 {
		host = default_ssh_host
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

	ssh := SSH{host: host, port: port, config: config}

	flag, err := ssh.check()
	if !flag {
		return errors.New(fmt.Sprintf("%s"+err.Error(), ErrNoSSH))
	}

	return nil
}

func AuthSSH(host, port, username, password string) error {
	if len(host) == 0 {
		host = default_ssh_host
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

	s := SSH{host: host, port: port, config: config}

	s.config.User = username
	s.config.Auth = []ssh.AuthMethod{ssh.Password(password)}

	client, err := ssh.Dial("tcp", s.host+":"+s.port, &s.config)
	if err != nil {
		return err
	}
	defer client.Close()

	return nil
}

func (s *SSH) AuthSSHRtries(username, password string) error {
	sum := 0
	for {
		if sum > s.rtries {
			return ErrRtries
		}
		err := s.authSSH(username, password)
		if err != nil && strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host") && sum <= s.rtries {
			sum++
			continue
		} else if err != nil && strings.Contains(err.Error(), "handshake failed: EOF") && sum <= s.rtries {
			sum++
			continue
		} else if err != nil {
			return err
		}
		return nil
	}
}

func (s *SSH) authSSH(username, password string) error {
	s.config.User = username
	s.config.Auth = []ssh.AuthMethod{ssh.Password(password)}

	client, err := ssh.Dial("tcp", s.host+":"+s.port, &s.config)
	if err != nil {
		return err
	}
	defer client.Close()

	return nil
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
