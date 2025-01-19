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
	"container/list"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/devsheke/scrapollo/internal/actions"
	"github.com/devsheke/scrapollo/internal/io"
	"github.com/devsheke/scrapollo/internal/models"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog/log"
)

var (
	ErrorDailyLimit    = errors.New("the daily limit for saving leads has been hit")
	ErrorNoCredits     = errors.New("no more credits available for saving leads")
	ErrorTargetReached = errors.New("target number of leads have been saved")
)

type browserWrapper struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
}

func newBrowserWrapper(headless bool) (*browserWrapper, error) {
	log.Debug().Msg("starting a new browser instance")

	wrapper := new(browserWrapper)
	if browserPath, ok := os.LookupEnv("BROWSER"); ok {
		wrapper.launcher = launcher.New().Bin(browserPath)
	} else {
		wrapper.launcher = launcher.New()
	}

	wrapper.launcher = wrapper.launcher.Headless(headless)

	controlURL, err := wrapper.launcher.Launch()
	if err != nil {
		return nil, err
	}
	wrapper.browser = rod.New().ControlURL(controlURL)

	return wrapper, wrapper.browser.Connect()
}

func (bw *browserWrapper) close() error {
	log.Debug().Msg("closing browser instance")

	if err := bw.browser.Close(); err != nil {
		return err
	}
	bw.launcher.Cleanup()

	log.Debug().Msg("closed browser instance and performed cleanup")

	return nil
}

const (
	progressFilePrefix     string = "scrapollo-progress"
	accountCookiesFilename string = "scrapollo-cookies.json"
)

func (r *Runner) _saveProgress() error {
	accs := make([]*models.Account, 0, r.jobs.Len())
	accCookies := make(map[string][]*proto.NetworkCookie, r.jobs.Len())

	for _, job := range r.jobs.iter() {
		if cookies, ok := job.acc.GetLoginCookies(); ok {
			accCookies[job.acc.Email] = cookies
		}
		accs = append(accs, job.acc)
	}

	cookiesFile := filepath.Join(r.outputDir, accountCookiesFilename)
	log.Debug().Str("file", cookiesFile).Msg("saving cookies")

	if err := io.SaveRecords(cookiesFile, accCookies); err != nil {
		return err
	}

	progressFile := filepath.Join(r.outputDir, progressFilePrefix+string(r.outputFormat))
	log.Debug().Str("file", progressFile).Msg("saving progress")

	return io.SaveRecords(progressFile, accs)
}

func (r *Runner) removeAnnoyances(page *rod.Page) error {
	for _, annoyance := range r.annoyances {
		if err := actions.RemoveAnnoyance(page, annoyance, r.timeout); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) newScrapingPage(page *rod.Page, bw *browserWrapper, acc *models.Account) error {
	log.Debug().Str("account", acc.Email).Msg("creating new scraping page")

	info, err := page.Info()
	if err != nil {
		return err
	}

	url := info.URL
	if err := page.Close(); err != nil {
		return err
	}

	newPage, err := actions.ApolloLogin(bw.browser, acc, r.stealth)
	*page = *newPage

	if err != nil {
		return err
	}

	err = page.Navigate(url)
	if err != nil {
		return err
	}

	log.Debug().Str("account", acc.Email).Msg("created and initialised new scraping page")

	return nil
}

func (r *Runner) scrapeLeads(page *rod.Page, bw *browserWrapper, job *job) error {
	file := filepath.Join(r.outputDir, job.acc.List+string(r.outputFormat))

	var writer io.LeadWriter
	switch r.outputFormat {
	case io.CsvFileFormat:
		writer = io.NewCsvLeadWriter(file)
	case io.JsonFileFormat:
		writer = io.NewJsonLeadWriter(file)
	}

	if err := r.removeAnnoyances(page); err != nil {
		return err
	}

	log.Info().Str("account", job.acc.Email).Msg("scraping leads")
	if err := actions.LocateList(page, job.acc.List, r.timeout); err != nil {
		return err
	}

	pageCount := 1
	total := 0
	for {
		if (pageCount-1) > 0 && (pageCount-1)%10 == 0 {
			if err := r.newScrapingPage(page, bw, job.acc); err != nil {
				return err
			}
		}

		if err := r.removeAnnoyances(page); err != nil {
			return err
		}

		pageData, err := actions.GetPageData(page, r.timeout)
		if err != nil {
			return err
		}

		if pageData.LastPage {
			return nil
		}

		leads, err := actions.ScrapeLeads(page, r.timeout)
		if err != nil {
			return err
		}
		total += len(leads)

		if err := writer.WriteLeads(leads); err != nil {
			log.Error().
				Err(err).
				Str("account", job.acc.Email).
				Msg("failed to write leads")
		}

		log.Info().Str("account", job.acc.Email).Int("num", total).Msg("scraped leads")

		switch err := pageData.NextPage(page); err {
		case nil:
			pageCount++
		case actions.ErrorListEnd:
			return nil
		default:
			return errors.Join(err, os.Remove(file))
		}
	}
}

func (r *Runner) saveLeads(job *job) (err error) {
	bw, err := newBrowserWrapper(r.headless)
	if err != nil {
		return err
	}
	defer bw.close()

	page, err := actions.ApolloLogin(bw.browser, job.acc, r.stealth)
	if err != nil {
		return err
	}

	defer func() {
		switch err {
		case nil, ErrorTargetReached, ErrorDailyLimit:
		default:
			if _err := actions.GrabErrorSnapshot(page, job.acc, r.errorDir); _err != nil {
				log.Warn().Err(err).Msg("failed to grab error snapshot")
			}
		}
	}()

	if r.fetchCredits {
		if err := r.removeAnnoyances(page); err != nil {
			return err
		}

		c, r, err := actions.FetchCreditUsage(page, job.acc, r.timeout)
		if err != nil {
			return err
		}

		job.acc.Credits, job.acc.CreditRefresh = c, r
	}

	if err = page.Navigate(job.acc.URL); err != nil {
		return err
	}

	if _, ok := job.startedAt.Get(); !ok {
		job.start()
	}

	if err := r.removeAnnoyances(page); err != nil {
		return err
	}

	if err := r.tab.Select(page); err != nil {
		return err
	}

	log.Debug().Str("tab", string(r.tab)).Msg("selected tab")

	var prevErr error
	var retries int
	for {
		if retries >= 5 {
			return prevErr
		}

		if job.acc.IsDone() {
			log.Info().
				Str("account", job.acc.Email).
				Str("list", job.acc.List).
				Msg("finished saving leads")

			if err = r.scrapeLeads(page, bw, job); err == nil {
				return
			}
			prevErr, retries = err, retries+1
			continue
		}

		if job.hitDailyLimit(r.limit) {
			return ErrorDailyLimit
		}

		if !job.acc.CanScrape() {
			return ErrorNoCredits
		}

		if err := r.removeAnnoyances(page); err != nil {
			return err
		}

		pageData, err := actions.GetPageData(page, r.timeout)
		if err != nil {
			return err
		}

		if err = actions.SaveLeads(page, job.acc.List, r.timeout); err != nil {
			prevErr, retries = err, retries+1
			continue
		}

		log.Info().
			Str("account", job.acc.Email).
			Str("list", job.acc.List).
			Int("page", pageData.Size).
			Msg("saved leads")

		job.incrementSaved(pageData.Size)

		if r.saveProgress {
			if err := r._saveProgress(); err != nil {
				log.Warn().Err(err).Msg("failed to save progress")
			}
		}

		if pageData.LastPage {
			job.acc.Target = job.acc.Saved
		}
	}
}

func (r *Runner) rearrangeJobs() {
	log.Debug().Msg("rearranging jobs")

	jobs := make([]*job, r.jobs.Len())
	for i, job := range r.jobs.iter() {
		jobs[i] = job
	}

	slices.SortFunc(jobs, func(a, b *job) int {
		timeoutA, okA := a.acc.Timeout.Get()
		timeoutB, okB := a.acc.Timeout.Get()

		if !okA && !okB {
			return 0
		}

		if !okA {
			return -1
		}

		if !okB {
			return 1
		}

		if timeoutA.Before(timeoutB) {
			return -1
		}

		if timeoutB.Before(timeoutA) {
			return 1
		}

		return 0
	})

	q := list.New()
	for _, job := range jobs {
		q.PushBack(job)
	}

	r.jobs.List = q

	log.Debug().Msg("rearranged jobs")
}

func unwrapError(err error) error {
	switch err := err.(type) {
	case *rod.TryError:
		return err.Unwrap()
	default:
		return err
	}
}

func (r *Runner) Start() error {
	var timeoutSkip int
	for {
		if r.jobs.isEmpty() {
			log.Info().Msg("finished all scraping jobs")
			break
		}

		_job, _ := r.jobs.Front().Value.(*job)
		acc := _job.acc

		if _, ok := acc.Timeout.Get(); ok {
			if timeoutSkip >= r.jobs.Len() {
				r.rearrangeJobs()
				_job, _ = r.jobs.Front().Value.(*job)
				if t, ok := _job.acc.Timeout.Get(); ok {
					dur := time.Until(t)
					log.Warn().Dur("duration", dur).Msg("pausing execution")
					time.Sleep(dur)
				}

				acc = _job.acc
				timeoutSkip = 0
			} else {
				timeoutSkip++
				if err := r.jobs.requeue(); err != nil {
					return err
				}
				continue
			}
		}

		switch err := r.saveLeads(_job); err {
		case ErrorDailyLimit:
			log.Warn().Str("account", acc.Email).Msg("hit daily save limit")
			acc.Timeout.Set(time.Now().Add(24 * time.Hour))
			if err := r.jobs.requeue(); err != nil {
				return err
			}

		case ErrorNoCredits:
			log.Warn().Str("account", acc.Email).Msg("out of credits")
			if err := r.jobs.requeue(); err != nil {
				return err
			}

		case actions.ErrorSecurityChallenge:
			log.Error().Err(err).Str("account", acc.Email).Msg("")

		case ErrorTargetReached, actions.ErrorListEnd:
			log.Info().Str("account", acc.Email).Msg("scraping completed")
			r.jobs.Remove(r.jobs.Front())

		default:
			log.Error().Err(unwrapError(err)).Str("account", acc.Email).Msg("scraping error")
			if err := r.jobs.requeue(); err != nil {
				return err
			}
		}

		if err := r._saveProgress(); err != nil {
			log.Error().Err(err).Msg("failed to save scraping progress")
		}
	}

	return nil
}
