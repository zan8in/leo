package leo

import "strings"

const (
	SSH_NAME  = "ssh"
	SSH_PORT  = "22"
	SSH_USERS = "root,kali"
	SSH_PWDS  = "root,kali"

	FTP_NAME  = "ftp"
	FTP_PORT  = "21"
	FTP_USERS = "test"
	FTP_PWDS  = "test"
)

type DefaultService struct {
	Port      string
	Users     []string
	Passwords []string
}

var DefaultServicePort = map[string]DefaultService{
	SSH_NAME: DefaultService{
		Port:      SSH_PORT,
		Users:     str2Slice(SSH_USERS),
		Passwords: str2Slice(SSH_PWDS),
	},
	FTP_NAME: DefaultService{
		Port:      FTP_PORT,
		Users:     str2Slice(FTP_USERS),
		Passwords: str2Slice(FTP_PWDS),
	},
}

func str2Slice(str string) []string {
	return strings.Split(str, ",")
}
