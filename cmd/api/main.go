package main

import (
	"fmt"

	"github.com/zan8in/leo/pkg/leo"
)

func main() {
	opt := leo.Options{
		Target: "mysql://192.168.3.25",
		// Target: "ssh://121.37.66.33",
		// User:         "root",
		// PasswordFile: "./p.txt",
	}

	if err := leo.NewOptionsApi(&opt); err != nil {
		panic(err)
	}

	r, err := leo.NewRunnerApi(&opt)
	if err != nil {
		panic(err)
	}

	if result := r.RunApi(); result != nil {
		fmt.Println("result:", result)
	}

}
