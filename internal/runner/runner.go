// Copyright 2025 Abhisheke Acharya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/devsheke/scrapollo/internal/actions"
	"github.com/devsheke/scrapollo/internal/io"
	"github.com/devsheke/scrapollo/internal/models"
	"github.com/go-rod/rod/lib/proto"
)

// Runner is a type that manages and orchestrates the process of scraping leads from Apollo.
type Runner struct {
	annoyances                                           []*actions.Annoyance
	debug, fetchCredits, headless, saveProgress, stealth bool
	jobs                                                 *queue
	limit                                                int
	outputFormat                                         io.FileFormat
	cookieFile, outputDir, errorDir                      string
	tab                                                  actions.ApolloTab
	timeout                                              time.Duration
}

const (
	bannerAnnoyance  string = "new-ui"
	newUiAnnoyance   string = "pop-up"
	popUpAnnoyances  string = "banner"
	sideNavAnnoyance string = "sidenav"
)

// RunnerOpt represents a function that is used to configure an instance of [Runner].
type RunnerOpt func(r *Runner)

// Annoyances is a [RunnerOpt] func that is used to specify which annoyances on Apollo
// to look out for.
func Annoyances(values []string) RunnerOpt {
	return func(r *Runner) {
		annoyances := []*actions.Annoyance{}
		for _, s := range values {
			switch s {
			case bannerAnnoyance:
				annoyances = append(annoyances, actions.TopBannerAnnoyance)
			case newUiAnnoyance:
				annoyances = append(annoyances, actions.NewUIAnnoyance)
			case popUpAnnoyances:
				annoyances = append(annoyances, actions.PopupDialogAnnoyance)
			case sideNavAnnoyance:
				annoyances = append(annoyances, actions.SidenavAnnoyance)
			}
		}
		r.annoyances = annoyances
	}
}

func CookieFile(file string) RunnerOpt {
	return func(r *Runner) {
		r.cookieFile = file
	}
}

// CsvOutput is a [RunnerOpt] func that sets the desired output format to CSV.
func CsvOutput() RunnerOpt {
	return func(r *Runner) {
		r.outputFormat = io.CsvFileFormat
	}
}

// Debug is a [RunnerOpt] func that configures the [Runner] to print useful
// debugging information.
func Debug(b bool) RunnerOpt {
	return func(r *Runner) {
		r.debug = b
	}
}

// FetchCredits is a [RunnerOpt] func that configures the [Runner] to fetch the
// credits for each [models.Account] before scraping.
func FetchCredits(b bool) RunnerOpt {
	return func(r *Runner) {
		r.fetchCredits = b
	}
}

// Tab is a [RunnerOpt] func that configures the [Runner] to scrape leads from
// the specified tab on Apollo.
func Tab(tab string) RunnerOpt {
	return func(r *Runner) {
		switch tab {
		case "new":
			r.tab = actions.NetNewTab
		case "saved":
			r.tab = actions.SavedTab
		case "total":
			r.tab = actions.TotalTab
		default:
			r.tab = actions.NetNewTab
		}
	}
}

// Headless is a [RunnerOpt] func that configures whether or not the [Runner] launches
// the browser in headless mode.
func Headless(b bool) RunnerOpt {
	return func(r *Runner) {
		r.headless = b
	}
}

// JsonOutput is a [RunnerOpt] func that sets the desired output format to CSV.
func JsonOutput() RunnerOpt {
	return func(r *Runner) {
		r.outputFormat = io.JsonFileFormat
	}
}

// Dailyimit is a [RunnerOpt] func that configures the [Runner]'s daily limit for saving leads on Apollo.
func Dailyimit(l int) RunnerOpt {
	return func(r *Runner) {
		r.limit = l
	}
}

// OutputDir is a [RunnerOpt] func that specifies the output directory for [Runner]'s output files.
func OutputDir(outputDir string) RunnerOpt {
	return func(r *Runner) {
		r.outputDir = outputDir
	}
}

// SaveProgress is a [RunnerOpt] func that specifies whether or not the [Runner] saves the intermediary state
// for each of the [models.Account]s.
func SaveProgress(b bool) RunnerOpt {
	return func(r *Runner) {
		r.saveProgress = b
	}
}

// Stealth is a [RunnerOpt] func that specifies whether or not the [Runner] launches the browser in stealth mode.
func Stealth(s bool) RunnerOpt {
	return func(r *Runner) {
		r.stealth = s
	}
}

// Timeout is a [RunnerOpt] func that configures the [Runner]'s time limit for each browser action.
func Timeout(t time.Duration) RunnerOpt {
	return func(r *Runner) {
		r.timeout = t
	}
}

// New returns a newly insantiated and configured instance of [Runner].
func New(accounts []*models.Account, opts ...RunnerOpt) (*Runner, error) {
	r := &Runner{
		limit:     500,
		timeout:   60 * time.Second,
		outputDir: "./apollo-output",
	}

	for _, optFn := range opts {
		optFn(r)
	}

	r.jobs = newQueue(accounts)
	for _, job := range r.jobs.iter() {
		if job.acc.CreditRefresh == nil {
			job.acc.CreditRefresh = &models.Time{}
		}

		if job.acc.Timeout == nil {
			job.acc.Timeout = &models.Time{}
		}
	}

	if r.cookieFile != "" {
		var accCookies map[string][]*proto.NetworkCookie
		if err := io.ReadRecords(r.cookieFile, &accCookies); err != nil {
			return nil, fmt.Errorf("failed to read cookie file: %v", err)
		}

		for _, job := range r.jobs.iter() {
			if cookies, ok := accCookies[job.acc.Email]; ok {
				job.acc.SetLoginCookies(cookies)
			}
		}
	}

	r.errorDir = filepath.Join(r.outputDir, "errors")
	if err := os.MkdirAll(r.errorDir, 0755); err != nil {
		return nil, err
	}

	return r, nil
}
