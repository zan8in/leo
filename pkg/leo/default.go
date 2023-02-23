package leo

import (
	"embed"
	"strings"
)

//go:embed dics/*
var f embed.FS

const (
	DEFAULT_PASS_FILE = "dics/default_pass.txt"

	SSH_NAME      = "ssh"
	SSH_PORT      = "22"
	SSH_USER_DICS = "dics/ssh_user.txt"
	SSH_PWDS_DICS = "dics/ssh_pass.txt"

	MYSQL_NAME      = "mysql"
	MYSQL_PORT      = "3306"
	MYSQL_USER_DICS = "dics/mysql_user.txt"
	MYSQL_PWDS_DICS = "dics/mysql_pass.txt"

	MSSQL_NAME      = "mssql"
	MSSQL_PORT      = "3306"
	MSSQL_USER_DICS = "dics/mssql_user.txt"
	MSSQL_PWDS_DICS = "dics/mssql_pass.txt"

	POSTGRES_NAME      = "postgres"
	POSTGRES_PORT      = "5432"
	POSTGRES_USER_DICS = "dics/postgres_user.txt"
	POSTGRES_PWDS_DICS = "dics/postgres_pass.txt"

	REDIS_NAME      = "redis"
	REDIS_PORT      = "6379"
	REDIS_USER_DICS = "dics/redis_user.txt"
	REDIS_PWDS_DICS = "dics/redis_pass.txt"

	FTP_NAME      = "ftp"
	FTP_PORT      = "21"
	FTP_USER_DICS = "dics/ftp_user.txt"
	FTP_PWDS_DICS = "dics/ftp_pass.txt"

	ORACLE_NAME      = "oracle"
	ORACLE_PORT      = "1521"
	ORACLE_USER_DICS = "dics/oracle_user.txt"
	ORACLE_PWDS_DICS = "dics/oracle_pass.txt"
)

const (
	STATUS_SUCCESS = 1
	STATUS_FAILED
	STATUS_COMPLATE
)

type DefaultService struct {
	Port      string
	Users     []string
	Passwords []string
}

var DefaultServicePort = map[string]DefaultService{
	SSH_NAME: {
		Port:      SSH_PORT,
		Users:     getDicsFromPath(SSH_USER_DICS),
		Passwords: getDicsFromPath(SSH_PWDS_DICS),
	},
	MYSQL_NAME: {
		Port:      MYSQL_PORT,
		Users:     getDicsFromPath(MYSQL_USER_DICS),
		Passwords: getDicsFromPath(MYSQL_PWDS_DICS),
	},
	MSSQL_NAME: {
		Port:      MSSQL_PORT,
		Users:     getDicsFromPath(MSSQL_USER_DICS),
		Passwords: getDicsFromPath(MSSQL_PWDS_DICS),
	},
	POSTGRES_NAME: {
		Port:      POSTGRES_PORT,
		Users:     getDicsFromPath(POSTGRES_USER_DICS),
		Passwords: getDicsFromPath(POSTGRES_PWDS_DICS),
	},
	REDIS_NAME: {
		Port:      REDIS_PORT,
		Users:     getDicsFromPath(REDIS_USER_DICS),
		Passwords: getDicsFromPath(REDIS_PWDS_DICS),
	},
	FTP_NAME: {
		Port:      FTP_PORT,
		Users:     getDicsFromPath(FTP_USER_DICS),
		Passwords: getDicsFromPath(FTP_PWDS_DICS),
	},
	ORACLE_NAME: {
		Port:      ORACLE_PORT,
		Users:     getDicsFromPath(ORACLE_USER_DICS),
		Passwords: getDicsFromPath(ORACLE_PWDS_DICS),
	},
}

func initPasswords() []string {
	return getDicsFromPath(DEFAULT_PASS_FILE)
}

func getDicsFromPath(path string) []string {
	var result []string

	file, err := f.ReadFile(path)
	if err != nil {
		return result
	}

	flist := strings.Split(string(file), "\r\n")
	result = append(result, flist...)

	return result
}
