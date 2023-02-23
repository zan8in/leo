package leo

import (
	"errors"

	"github.com/zan8in/leo/pkg/ftp"
	"github.com/zan8in/leo/pkg/mssql"
	"github.com/zan8in/leo/pkg/mysql"
	"github.com/zan8in/leo/pkg/oracle"
	"github.com/zan8in/leo/pkg/postgres"
	"github.com/zan8in/leo/pkg/redis"
	"github.com/zan8in/leo/pkg/ssh"
)

type Execute struct {
	options *Options
}

func NewExecute(options *Options) *Execute {
	return &Execute{options: options}
}

func (e *Execute) start(host, username, password string, m any) error {
	service := e.options.Service
	if service == SSH_NAME {
		client := m.(*ssh.SSH)
		return client.AuthRtries(host, username, password)
	}
	if service == MYSQL_NAME {
		client := m.(*mysql.MYSQL)
		return client.AuthRetries(username, password)
	}
	if service == MSSQL_NAME {
		client := m.(*mssql.MSSQL)
		return client.AuthRetries(username, password)
	}
	if service == POSTGRES_NAME {
		client := m.(*postgres.POSTGRES)
		return client.AuthRetries(username, password)
	}
	if service == FTP_NAME {
		client := m.(*ftp.FTP)
		return client.AuthRetries(username, password)
	}
	if service == REDIS_NAME {
		client := m.(*redis.REDIS)
		return client.AuthRetries(username, password)
	}
	if service == ORACLE_NAME {
		client := m.(*oracle.ORACLE)
		return client.AuthRetries(username, password)
	}
	return nil
}

func (e *Execute) validateService(host, port string) (any, error) {
	service := e.options.Service
	if service == SSH_NAME {
		m, err := ssh.New(host, port, e.options.Retries, e.options.Timeout)
		if err != nil {
			return m, err
		}
		return m, nil
	}
	if service == MYSQL_NAME {
		m, err := mysql.New(host, port, e.options.Retries, e.options.Timeout)
		if err != nil {
			return m, err
		}
		return m, nil
	}
	if service == MSSQL_NAME {
		m, err := mssql.New(host, port, e.options.Retries, e.options.Timeout)
		if err != nil {
			return m, err
		}
		return m, nil
	}
	if service == POSTGRES_NAME {
		m, err := postgres.New(host, port, e.options.Retries, e.options.Timeout)
		if err != nil {
			return m, err
		}
		return m, nil
	}
	if service == FTP_NAME {
		m, err := ftp.New(host, port, e.options.Retries, e.options.Timeout)
		if err != nil {
			return m, err
		}
		return m, nil
	}
	if service == REDIS_NAME {
		m, err := redis.New(host, port, e.options.Retries, e.options.Timeout)
		if err != nil {
			return m, err
		}
		return m, nil
	}
	if service == ORACLE_NAME {
		m, err := oracle.New(host, port, e.options.Retries, e.options.Timeout)
		if err != nil {
			return m, err
		}
		return m, nil
	}

	return nil, errors.New("validdate service failed.")
}
