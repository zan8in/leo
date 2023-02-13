package leo

import (
	"fmt"

	"github.com/zan8in/gologger"
)

var Version = "0.1.0"

var banner = fmt.Sprintf(`
    _/        _/_/_/_/    _/_/    
   _/        _/        _/    _/   
  _/        _/_/_/    _/    _/    
 _/        _/        _/    _/     
_/_/_/_/  _/_/_/_/    _/_/    %s
`, Version)

func ShowBanner() {
	gologger.Print().Msgf("%s\n", banner)
	gologger.Print().Msgf("\t\t\tThe Wandering Earth 2\n\n")
}
