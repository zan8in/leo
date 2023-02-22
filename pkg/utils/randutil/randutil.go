package randutil

import (
	"math/rand"
	"time"
)

// 包含上下限 [min, max]
func GetRandomIntWithAll(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return int(rand.Intn(max-min+1) + min)
}

// 不包含上限 [min, max)
func GetRandomIntWithMin(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return int(rand.Intn(max-min) + min)
}
