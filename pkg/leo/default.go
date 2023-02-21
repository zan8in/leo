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
