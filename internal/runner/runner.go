package runner

import (
	"os"
	"path/filepath"
	"time"

	"github.com/shadowbizz/apollo-crawler/internal/actions"
	"github.com/shadowbizz/apollo-crawler/internal/io"
	"github.com/shadowbizz/apollo-crawler/internal/models"
	"github.com/shadowbizz/apollo-crawler/internal/openvpn"
)

// ScrapeRunner is a job queue which orchestrates and controls the execution
// the scrape jobs alotted to each of the given apollo.io accounts.
type ScrapeRunner struct {
	debug, fetchCredits, headless, saveProgress bool
	limit, outputKind                           int
	jobs                                        *JobQueue
	outputDir, errorDir, tab                    string
	timeout                                     time.Duration
	vpn                                         *openvpn.OpenVPN
	leadWriters                                 []io.LeadWriter
}

// RunnerOpt is a function which is used to configure the apollo.io scrape Queue.
type RunnerOpt func(r *ScrapeRunner)

// CSVOutput sets the output to CSV format.
func CSVOutput() RunnerOpt {
	return func(r *ScrapeRunner) {
		r.outputKind = io.CSVOutput
	}
}

// Debug runs the Queue in debug mode.
//
// In this mode error screenshots are captured to better debug any problems faced
// during the scrape
func Debug(b bool) RunnerOpt {
	return func(r *ScrapeRunner) {
		r.debug = b
	}
}

// FetchCredits specifies if the credits are to be fetched for each of the given initAccounts
// before the scrape jobs are started.
func FetchCredits(b bool) RunnerOpt {
	return func(r *ScrapeRunner) {
		r.fetchCredits = b
	}
}

// SetTab is used to specify that each scraper scrapes detail from the given apollo tab.
func SetTab(tab string) RunnerOpt {
	return func(r *ScrapeRunner) {
		switch tab {
		case "new":
			r.tab = actions.NetNewTab
		case "saved":
			r.tab = actions.SavedTab
		case "total":
			r.tab = actions.TotalTab
		}
	}
}

// Headless specifies whether Chrome should run in headless mode or not.
func Headless(b bool) RunnerOpt {
	return func(r *ScrapeRunner) {
		r.headless = b
	}
}

// JSONOutput sets the output to JSON format.
func JSONOutput() RunnerOpt {
	return func(r *ScrapeRunner) {
		r.outputKind = io.JSONOutput
	}
}

// Dailyimit sets the maximum number of records that can be scraped per job, per day.
// The default value for this option is 500.
func Dailyimit(l int) RunnerOpt {
	return func(r *ScrapeRunner) {
		r.limit = l
	}
}

// OutputDir specifies the directory which will be used to store all the relevant output files.
func OutputDir(outputDir string) RunnerOpt {
	return func(r *ScrapeRunner) {
		r.outputDir = outputDir
	}
}

// SaveProgress specifies if intermediary job progress should be stored.
// The progress data includes the total saved leads and credit information.
func SaveProgress(b bool) RunnerOpt {
	return func(r *ScrapeRunner) {
		r.saveProgress = b
	}
}

// Timeout sets the maximum amount of time (in seconds) that can be spent on a chrome automation action.
func Timeout(t time.Duration) RunnerOpt {
	return func(r *ScrapeRunner) {
		r.timeout = t
	}
}

// VPN specifies that the Queue run with OpenVPN.
func VPN(o *openvpn.OpenVPN) RunnerOpt {
	return func(r *ScrapeRunner) {
		r.vpn = o
	}
}

// New instantiates a new Queue instance with the given apollo.io accounts as well as QueueOpts.
func New(accounts []*models.ApolloAccount, opts ...RunnerOpt) (*ScrapeRunner, error) {
	r := &ScrapeRunner{
		limit:     500,
		timeout:   60 * time.Second,
		outputDir: "./apollo-output",
	}

	for _, optFn := range opts {
		optFn(r)
	}

	r.jobs = NewJobQueue(accounts)
	for job := range r.jobs.Iter {
		if r.vpn != nil && job.account.VpnConfig == "" {
			for _, config := range r.vpn.Configs {
				if !r.vpn.ConfigIsUsed(config) {
					job.account.VpnConfig = config
				}
			}
		}
	}

	r.errorDir = filepath.Join(r.outputDir, "errors")
	if err := os.MkdirAll(r.errorDir, 0755); err != nil {
		return nil, err
	}

	return r, nil
}
