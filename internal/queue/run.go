package queue

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/internal/actions"
	"github.com/shadowbizz/apollo-crawler/internal/io"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

var (
	// ErrorTargetReached indicates that the scrape target set for an ApolloAccount
	// has been successfully reached.
	ErrorTargetReached = errors.New("scrape target has been reached")

	// ErrorTargetReached indicates that the the daily limit for scraping an ApolloAccount
	// has been reached.
	ErrorDailyLimit = errors.New("scrape daily limit has been reachehd")
)

// fetchCredits fetches and updates the credits for each ApolloAccount present
// in the queue.
func (q *Queue) _fetchCredits() error {
	for _, job := range q.jobs {
		ctx, cancel := q.newChromeContext()

		if err := actions.SignIn(ctx, job.account.Email, job.account.Password, q.timeout); err != nil {
			cancel()
			return err
		}

		current, refresh, err := actions.FetchCredits(ctx, q.timeout)
		if err != nil {
			cancel()
			return err
		}

		job.account.Credits = current
		job.account.CreditRefresh = models.NewTime(refresh)

		cancel()
	}

	return nil
}

// _saveProgress saves the intermediary state of each ApolloAccount present
// in the queue.
func (q *Queue) _saveProgress() error {
	accounts := make([]*models.ApolloAccount, len(q.jobs))
	for i, job := range q.jobs {
		accounts[i] = job.account
	}

	ext, err := io.ExtensionFromOutputType(io.CSVOutput)
	if err != nil {
		return err
	}

	return io.SaveRecordsToFile(
		accounts,
		filepath.Join(
			q.outputDir,
			filepath.Join(q.outputDir, fmt.Sprintf("apollo-scrape-progress%s", ext)),
		),
		io.CSVOutput,
	)
}

// scrapeJob runs a single isolated task of scraping leads from the url allocated
// to each ApolloAccount until the daily limit has been reached or until completed.
func (q *Queue) scrapeJob(job *job) error {
	defer func() {
		if q.vpn != nil {
			if err := q.vpn.Stop(); err != nil {
				log.Warn().Err(err).Msg("openvpn stop err")
			}
		}
	}()

	if q.vpn != nil {
		if err := q.vpn.Start(job.account.VpnConfig); err != nil {
			newConf, err := q.vpn.Backup()
			if err != nil {
				return err
			}
			job.account.VpnConfig = newConf
		}
	}

	ctx, cancel := q.newChromeContext()
	defer cancel()

	if err := actions.SignIn(ctx, job.account.Email, job.account.Password, q.timeout); err != nil {
		return err
	}

	if err := chromedp.Run(ctx, chromedp.Navigate(job.account.URL)); err != nil {
		return err
	}

	if q.tab != "" {
		if err := actions.SetTab(ctx, q.tab, q.timeout); err != nil {
			return err
		}
	}

	var lastErr error
	var leads []*models.ApolloLead
	var retries int

	for len(leads) <= q.limit {
		if retries >= 10 {
			return lastErr
		}

		_leads, err := actions.ScrapePage(ctx, q.timeout)
		if err != nil {
			retries++
			lastErr = err
			continue
		}
		leads = append(leads, _leads...)

		for i := range q.leadWriters {
			if err := q.leadWriters[i].WriteLeads(leads); err != nil {
				log.Error().
					Err(err).
					Str("kind", q.leadWriters[i].Kind()).
					Msg("failed to write leads")
			}
		}

		if err != nil {
			retries++
			lastErr = err
			continue
		}

		job.account.IncSaved(len(leads))
		job.account.UseCredits(len(leads))

		// TODO: verbose credit usage tracking!

		if q.saveProgress {
			if err := q._saveProgress(); err != nil {
				log.Warn().Err(err).Msg("failed to save scraper progress")
			}
		}

		if job.account.IsDone() {
			return ErrorTargetReached
		}

		if err := actions.NextPage(ctx, q.timeout); err != nil {
			if errors.Is(err, actions.ErrorListEnd) {
				return err
			}
			retries++
			lastErr = err
			continue
		}
	}

	if len(leads) >= q.limit {
		job.account.SetTimeout(24 * time.Hour)
	}

	return ErrorDailyLimit
}

// Run executes the queue of scrape jobs till completion (i.e., no more jobs are available).
func (q *Queue) Run() error {
	if q.fetchCredits {
		if err := q._fetchCredits(); err != nil {
			return err
		}
	}

	for {
		job, err := q.front()
		if err != nil {
			log.Info().Msg("finished all scraping jobs")
			break
		}

		account := job.account

		if account.IsDone() {
			log.Info().Str("account", account.Email).Msg("scraper job complete")
			_, _ = q.dequeueJob()
			continue
		}

		if account.IsTimedOut() {
			log.Debug().
				Str("account", account.Email).
				Time("till", account.Timeout.Get()).
				Msg("skipping timed out job")

			_ = q.sendToBack()
			continue
		}

		err = q.scrapeJob(job)
		if errors.Is(err, ErrorDailyLimit) {
			log.Info().Str("account", account.Email).Msg("scraper job daily limit reached")
			_ = q.sendToBack()
		} else if errors.Is(err, ErrorTargetReached) || errors.Is(err, actions.ErrorListEnd) {
			log.Info().Str("account", account.Email).Msg("scraper job complete")
			_, _ = q.dequeueJob()
		} else {
			log.Error().Err(err).Str("account", account.Email).Msg("scraper job failed")
			_ = q.sendToBack()
		}
		continue
	}

	return nil
}
