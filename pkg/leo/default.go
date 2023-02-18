package leo

import (
	"embed"
	"strings"
)

//go:embed dics/*
var f embed.FS

const (
	SSH_NAME      = "ssh"
	SSH_PORT      = "22"
	SSH_USER_DICS = "dics/ssh_user.txt"
	SSH_PWDS_DICS = "dics/ssh_pass.txt"

	FTP_NAME      = "ftp"
	FTP_PORT      = "21"
	FTP_USER_DICS = "dics/ftp_user.txt"
	FTP_PWDS_DICS = "dics/ftp_pass.txt"
)

const (
	STATUS_SUCCESS = 1
	STATUS_FAILED
)

type DefaultService struct {
	Port      string
	Users     []string
	Passwords []string
}

var DefaultServicePort = map[string]DefaultService{
	SSH_NAME: DefaultService{
		Port:      SSH_PORT,
		Users:     getDicsFromPath(SSH_USER_DICS),
		Passwords: getDicsFromPath(SSH_PWDS_DICS),
	},
	FTP_NAME: DefaultService{
		Port:      FTP_PORT,
		Users:     getDicsFromPath(FTP_USER_DICS),
		Passwords: getDicsFromPath(FTP_PWDS_DICS),
	},
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
