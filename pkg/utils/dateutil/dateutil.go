package dateutil

import "time"

func GetNowFullDateTime() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05")
}

func GetNowDateTime() string {
	now := time.Now()
	return now.Format("01-02 15:04:05")
}
