package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/zan8in/gologger"
	"github.com/zan8in/leo/pkg/leo"
)

func main() {

	options := leo.ParseOptions()

	runner, err := leo.NewRunner(options)
	if err != nil {
		gologger.Fatal().Msg(err.Error())
	}

	starttime := time.Now()
	green := color.Green.Render

	runner.Run(func(a any) {
		result := a.(*leo.CallbackInfo)

		if result.Err == nil {
			gologger.Print().Msgf("\r[%s][%s][%s] username: %s password: %s\r\n", green(options.Service), green(result.Host), green(options.Port), green(result.Username), green(result.Password))
		} else {
			if result.Status == leo.STATUS_FAILED {
				gologger.Error().Msgf("[%s][%s][%s] Connection failed, %s\r\n", options.Service, result.Host, options.Port, result.Err.Error())
			} else {
				gologger.Debug().Msgf("\r[%s][%s][%s] username: %s password: %s, %s\r\n", options.Service, result.Host, options.Port, result.Username, result.Password, result.Err.Error())
			}
		}

		if !options.Silent {
			fmt.Printf("\r%d/%d/%d%%/%s", result.CurrentCount, options.Count, result.CurrentCount*100/options.Count, strings.Split(time.Since(starttime).String(), ".")[0]+"s")
		}

	})

}
