package main

import (
	"github.com/zan8in/gologger"
	"github.com/zan8in/leo/pkg/leo"
)

func main() {

	options := leo.ParseOptions()

	runner, err := leo.NewRunner(options)
	if err != nil {
		gologger.Fatal().Msg(err.Error())
	}

	runner.Run()

	runner.Listener()
}
