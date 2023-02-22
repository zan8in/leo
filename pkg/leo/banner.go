package leo

import (
	"fmt"

	"github.com/zan8in/leo/pkg/utils/randutil"

	"github.com/zan8in/gologger"
)

var Version = "0.1.0"

var banner1 = fmt.Sprintf(`
┬  ┌─┐┌─┐
│  ├┤ │ │
┴─┘└─┘└─┘ %s
`, Version)
var banner2 = fmt.Sprintf(`
╦  ╔═╗╔═╗
║  ║╣ ║ ║
╩═╝╚═╝╚═╝ %s
`, Version)

func ShowBanner() {
	gologger.Print().Msgf("%s\n", randomBanner())
	gologger.Print().Msgf("\thttps://github.com/zan8in/leo\n\n")
}

func randomBanner() string {
	switch randutil.GetRandomIntWithAll(1, 2) {
	case 1:
		return banner1
	}
	return banner1
}
