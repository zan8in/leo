package main

import (
	"fmt"

	"github.com/zan8in/leo/pkg/leo"
)

func main() {

	options := leo.ParseOptions()

	runner, err := leo.NewRunner(options)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = runner.Run()
	if err != nil {
		fmt.Println(err.Error())
	}

	// fmt.Println(options)

}
