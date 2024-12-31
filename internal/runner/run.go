package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/internal/actions"
	"github.com/shadowbizz/apollo-crawler/internal/io"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

var (
	// ErrorTargetReached indicates that the scrape target set for an ApolloAccount
	// has been successfully reached.
	ErrorTargetReached = errors.New("scraper target has been reached")

	// ErrorTargetReached indicates that the daily limit for scraping an ApolloAccount
	// has been reached.
	ErrorDailyLimit = errors.New("scraper daily limit has been reached")

	// ErrorNoCredits indicates that the scraper has no more credits left to save leads.
	ErrorNoCredits = errors.New("scraper credits have been exhauster")
)

var screenOverride = func() *proto.EmulationSetDeviceMetricsOverride {
	width, height := 1920, 1080
	return &proto.EmulationSetDeviceMetricsOverride{
		DeviceScaleFactor: 2,
		Width:             width,
		Height:            height,
		ScreenWidth:       &width,
		ScreenHeight:      &height,
		Mobile:            false,
	}
}

// browserWrapper is an abstraction over the rod launcher and browser types.
type browserWrapper struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
}

// newBrowserWrapper constructs an instance of *browserWrapper and launches a new browser instance.
// This function checks the environment for a 'BROWSER' variable with a path a browser. If not found, rod
// takes over.
func newBrowserWrapper(headless bool) (*browserWrapper, error) {
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

func (b *browserWrapper) close() {
	if err := b.browser.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close browser")
	}
	b.launcher.Cleanup()
}

func removeSideNav(page *rod.Page) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	el, err := page.Context(ctx).Element("#side-nav")
	if errors.Is(err, &rod.ElementNotFoundError{}) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}

	return el.Remove()
}

// _saveProgress saves the intermediary state of each ApolloAccount present
// in the queue.
func (q *ScrapeRunner) _saveProgress() error {
	accounts := make([]*models.ApolloAccount, q.jobs.Len())

	var ptr int
	for job := range q.jobs.Iter {
		accounts[ptr] = job.account
		ptr++
	}

	ext, err := io.ExtensionFromOutputType(io.CSVOutput)
	if err != nil {
		return err
	}

	return io.SaveRecordsToFile(
		accounts,
		filepath.Join(
			q.outputDir,
			fmt.Sprintf("apollo-scrape-progress%s", ext),
		),
		io.CSVOutput,
	)
}

func (q *ScrapeRunner) scrapeLeads(page *rod.Page, bw *browserWrapper, job *ScrapeJob) error {
	ext, err := io.ExtensionFromOutputType(q.outputKind)
	if err != nil {
		return err
	}

	var file string
	if job.account.SaveToList != "" {
		file = filepath.Join(q.outputDir, job.account.SaveToList+ext)
	} else {

		file = filepath.Join(q.outputDir, job.outputName+ext)
	}

	var writer io.LeadWriter
	switch q.outputKind {
	case io.CSVOutput:
		writer = io.NewCSVLeadWriter(file)
	case io.JSONOutput:
		writer = io.NewJSONLeadWriter(file)
	}

	log.Info().Str("account", job.account.Email).Msg("scraping leads")
	if err := actions.GoToList(page, job.account.SaveToList, q.timeout); err != nil {
		return err
	}

	pageCount := 1
	total := 0
	for {
		if (pageCount-1) > 0 && (pageCount-1)%10 == 0 {
			info, err := page.Info()
			if err != nil {
				return err
			}

			url := info.URL
			if err := page.Close(); err != nil {
				return err
			}

			_page, err := actions.LoginToApollo(bw.browser, job.account, q.errorDir, q.timeout)
			if err != nil {
				return err
			}

			*page = *_page

			err = page.Navigate(url)
			if err != nil {
				return err
			}
		}

		err = actions.CloseNewUIDialog(page, 1*time.Second)
		if err != nil {
			return err
		}

		err = actions.ClosePowerUpDialog(page, 1*time.Second)
		if err != nil {
			return err
		}

		leads, err := actions.ScrapeLeads(page, q.tab, q.timeout)
		if err != nil {
			return err
		}

		for _, lw := range q.leadWriters {
			if err := lw.WriteLeads(leads); err != nil {
				log.Error().Err(err).Str("writer", lw.Kind()).Msg("failed to write to lead writer")
			}
		}

		// TODO: save leads that failed to be written for another try!
		if err := writer.WriteLeads(leads); err != nil {
			log.Error().
				Err(err).
				Str("account", job.account.Email).
				Msg("failed to save leads")
		}

		total += len(leads)

		log.Info().Str("account", job.account.Email).Int("num", total).Msg("scraped leads")

		if err := actions.GoToNextPage(page); err != nil {
			if !(errors.Is(err, actions.ErrorListEnd)) {
				err = errors.Join(err, os.Remove(file))
			}
			return err
		}

		pageCount++
	}
}

func (q *ScrapeRunner) saveLeads(job *ScrapeJob) (err error) {
	defer func() {
		if q.vpn != nil {
			if err := q.vpn.Stop(); err != nil {
				log.Warn().Err(err).Msg("openvpn stop err")
			}
		}
	}()

	if q.vpn != nil {
		if err := q.vpn.Restart(job.account.VpnConfig); err != nil {
			newConf, _err := q.vpn.Backup()
			if _err != nil {
				return errors.Join(err, _err)
			}
			job.account.VpnConfig = newConf
		}
	}

	bw, err := newBrowserWrapper(q.headless)
	if err != nil {
		return err
	}
	log.Info().Msg("allocated a new browser")

	page, err := actions.LoginToApollo(bw.browser, job.account, q.errorDir, q.timeout)
	if err != nil {
		return
	}

	if err := page.SetViewport(screenOverride()); err != nil {
		return err
	}

	defer func() {
		switch err {
		case ErrorTargetReached, actions.ErrorListEnd, ErrorDailyLimit:
			bw.close()
			return
		default:
			if err != nil && page != nil {
				_err := rod.Try(func() {
					file := filepath.Join(q.errorDir, job.account.Email)
					page.MustScreenshot(file + ".png")
					if err := os.WriteFile(file+".html", []byte(page.MustHTML()), 0644); err != nil {
						panic(err)
					}
				})
				if _err != nil {
					log.Error().Err(_err).Msg("failed to grab error screenshot")
				}
				err = errors.Join(err, _err)
			}

			bw.close()
		}
	}()

	if q.fetchCredits {
		err := actions.CloseNewUIDialog(page, 5*time.Second)
		if err != nil {
			return err
		}

		err = actions.ClosePowerUpDialog(page, 5*time.Second)
		if err != nil {
			return err
		}

		credits, refresh, err := actions.FetchCreditUsage(
			page,
			job.account,
			q.timeout,
		)
		if err != nil {
			return err
		}

		job.account.Credits, job.account.CreditRefresh = credits, refresh
		log.Info().
			Str("account", job.account.Email).
			Int("credits", credits).
			Time("renewal time", refresh.Get()).
			Msg("got credit info")
	}

	err = page.Navigate(job.account.URL)
	if err != nil {
		return err
	}

	if q.tab == "" {
		q.tab = actions.NetNewTab
	}

	var lastErr error
	var retries int

	log.Info().
		Str("account", job.account.Email).
		Str("list", job.account.SaveToList).
		Msg("saving leads to list")

	if !job.startedAt.IsSome() {
		job.startedAt.Set(time.Now())
	}

	err = rod.Try(func() {
		ctx, cancel := context.WithTimeout(context.Background(), q.timeout)
		defer cancel()

		page.Context(ctx).MustElement(".zp_tFLCQ .zp_hWv1I").MustWaitVisible()
	})
	if err != nil {
		return err
	}

	err = actions.CloseNewUIDialog(page, 5*time.Second)
	if err != nil {
		return err
	}

	err = actions.ClosePowerUpDialog(page, 5*time.Second)
	if err != nil {
		return err
	}

	if err := removeSideNav(page); err != nil {
		return err
	}

	if err := actions.SelectTab(page, q.tab); err != nil {
		return err
	}

	for {
		if retries >= 10 {
			return lastErr
		}

		if job.account.IsDone() {
			log.Info().
				Str("account", job.account.Email).
				Str("list", job.account.SaveToList).
				Msg("done saving leads")
			err := q.scrapeLeads(page, bw, job)
			if err != nil {
				return err
			}

			lastErr = err
			retries++
			continue
		}

		if !job.account.CanScrape() {
			return ErrorNoCredits
		}

		if job.IsDoneForToday(q.limit) {
			return ErrorDailyLimit
		}

		err = actions.CloseNewUIDialog(page, 1*time.Second)
		if err != nil {
			return err
		}

		err = actions.ClosePowerUpDialog(page, 1*time.Second)
		if err != nil {
			return err
		}

		numLeads, err := actions.SaveLeadsToList(
			page,
			job.account.SaveToList,
			q.timeout,
		)

		if errors.Is(err, actions.ErrorListEnd) {
			log.Info().
				Str("account", job.account.Email).
				Str("list", job.account.SaveToList).
				Msg("done saving leads")
			return q.scrapeLeads(page, bw, job)
		}

		if err != nil {
			retries++
			lastErr = err
			continue
		}

		job.account.IncSaved(numLeads)
		job.account.UseCredits(numLeads)
		job.saved += numLeads

		log.Info().
			Str("account", job.account.Email).
			Str("list", job.account.SaveToList).
			Int("num", job.account.Saved).
			Msg("saved leads")

		if q.saveProgress {
			if err := q._saveProgress(); err != nil {
				log.Warn().Err(err).Msg("failed to save scraper progress")
			}
			log.Info().Str("account", job.account.Email).Msg("saved scraper progress")
		}
	}
}

// Run executes the queue of scrape jobs till completion (i.e., no more jobs are available).
func (r *ScrapeRunner) Run() error {
	for {
		if r.jobs.IsEmpty() {
			log.Info().Msg("finished all scraping jobs")
			break
		}

		job, _ := r.jobs.Front().Value.(*ScrapeJob)
		account := job.account

		if account.IsTimedOut() {
			log.Info().
				Str("account", account.Email).
				Time("till", account.Timeout.Get()).
				Msg("skipping timed out job")

			if err := r.jobs.Requeue(); err != nil {
				return err
			}
			continue
		}

		err := r.saveLeads(job)
		switch err {
		case ErrorDailyLimit:
			log.Warn().Str("account", account.Email).Msg("scraper job daily limit reached")
			job.account.SetTimeout(24 * time.Hour)
			if err := r.jobs.Requeue(); err != nil {
				return err
			}

		case ErrorNoCredits:
			log.Warn().Str("account", account.Email).Msg("scraper has no more credits left")
			job.account.SetTimeout(time.Until(job.account.Timeout.Get()))
			if err := r.jobs.Requeue(); err != nil {
				return err
			}

		case ErrorTargetReached, actions.ErrorListEnd:
			log.Info().Str("account", account.Email).Msg("scraper job complete")
			r.jobs.Remove(r.jobs.Front())

		default:
			if err != nil {
				log.Error().
					Err(err).
					Str("account", account.Email).
					Msg("scraper encountered an error")
			}
			if err := r.jobs.Requeue(); err != nil {
				return err
			}
		}
		if err = r._saveProgress(); err != nil {
			log.Error().Err(err).Msg("failed to save scrapers' progress")
		}
	}

	return nil
}
