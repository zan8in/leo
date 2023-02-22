package dateutil

import "time"

func GetNowDateTime() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05")
}
