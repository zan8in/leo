package leo

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/tj/go-update"
	"github.com/tj/go-update/progress"
	githubUpdateStore "github.com/tj/go-update/stores/github"
	"github.com/zan8in/gologger"
)

func UpdateVersionToLatest(verbose bool) error {
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "leo.exe"
	default:
		command = "leo"
	}
	m := &update.Manager{
		Command: command,
		Store: &githubUpdateStore.Store{
			Owner:   "zan8in",
			Repo:    "leo",
			Version: Version,
		},
	}
	releases, err := m.LatestReleases()
	if err != nil {
		return errors.Wrap(err, "could not fetch latest release")
	}
	if len(releases) == 0 {
		gologger.Info().Msgf("No new updates found for Leo engine!")
		return nil
	}

	latest := releases[0]
	var currentOS string
	switch runtime.GOOS {
	case "darwin":
		currentOS = "macOS"
	default:
		currentOS = runtime.GOOS
	}
	final := latest.FindTarball(currentOS, runtime.GOARCH)
	if final == nil {
		return fmt.Errorf("no compatible binary found for %s/%s", currentOS, runtime.GOARCH)
	}
	final.URL = strings.Replace(final.URL, "github.com/zan8in", "gitee.com/zanbin", -1)
	tarball, err := final.DownloadProxy(progress.Reader)
	if err != nil {
		return errors.Wrap(err, "could not download latest release")
	}
	if err := m.Install(tarball); err != nil {
		return errors.Wrap(err, "could not install latest release")
	}
	gologger.Info().Msgf("Successfully updated to Leo %s\n", latest.Version)
	return nil
}
