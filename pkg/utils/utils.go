package utils

import "time"

func IsNumeric(s string) bool {
	for _, c := range s {
		if !(c >= 48 && c <= 57) {
			return false
		}
	}
	return true
}

func GetNowDateTime() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05")
}
