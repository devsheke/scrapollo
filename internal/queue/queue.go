package queue

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/shadowbizz/apollo-crawler/internal/actions"
	"github.com/shadowbizz/apollo-crawler/internal/io"
	"github.com/shadowbizz/apollo-crawler/internal/models"
	"github.com/shadowbizz/apollo-crawler/internal/openvpn"
)

var ErrorEmptyQueue = errors.New("job queue is empty")

// job represents a scrape task. It includes an apollo.io account
// as well as an output destination for the scrape data.
type job struct {
	output  string
	account *models.ApolloAccount
}

// Queue is a job queue which orchestrates and controls the execution
// the scrape jobs alotted to each of the given apollo.io accounts.
type Queue struct {
	debug, fetchCredits, headless, saveProgress bool
	limit                                       int
	jobs                                        []*job
	outputDir, tab                              string
	timeout                                     time.Duration
	vpn                                         *openvpn.OpenVPN
	leadWriters                                 []io.LeadWriter
}

// QueueOpt is a function which is used to configure the apollo.io scrape Queue.
type QueueOpt func(q *Queue)

// CSVOutput sets the output to CSV format.
//
// # NOTE: Make sure to call this function only after queue.outputDir has been set.
func CSVOutput() QueueOpt {
	return func(q *Queue) {
		w := io.NewCSVLeadWriter(q.getOutfileName(".csv"))
		q.leadWriters = append(q.leadWriters, w)
	}
}

// Debug runs the Queue in debug mode.
//
// In this mode error screenshots are captured to better debug any problems faced
// during the scrape
func Debug(b bool) QueueOpt {
	return func(q *Queue) {
		q.debug = b
	}
}

// FetchCredits specifies if the credits are to be fetched for each of the given initAccounts
// before the scrape jobs are started.
func FetchCredits(b bool) QueueOpt {
	return func(q *Queue) {
		q.fetchCredits = b
	}
}

// SetTab is used to specify that each scraper scrapes detail from the given apollo tab.
func SetTab(tab string) QueueOpt {
	return func(q *Queue) {
		switch tab {
		case "new":
			q.tab = actions.NetNewTab
		case "saved":
			q.tab = actions.SavedTab
		case "total":
			q.tab = actions.TotalTab
		}
	}
}

// Headless specifies whether Chrome should run in headless mode or not.
func Headless(b bool) QueueOpt {
	return func(q *Queue) {
		q.headless = b
	}
}

// JSONOutput sets the output to JSON format.
//
// # NOTE: Make sure to call this function only after queue.outputDir has been set.
func JSONOutput() QueueOpt {
	return func(q *Queue) {
		w := io.NewJSONLeadWriter(q.getOutfileName(".json"))
		q.leadWriters = append(q.leadWriters, w)
	}
}

// Dailyimit sets the maximum number of records that can be scraped per job, per day.
// The default value for this option is 500.
func Dailyimit(l int) QueueOpt {
	return func(q *Queue) {
		q.limit = l
	}
}

// OutputDir specifies the directory which will be used to store all the relevant output files.
func OutputDir(outputDir string) QueueOpt {
	return func(q *Queue) {
		q.outputDir = outputDir
	}
}

// SaveProgress specifies if intermediary job progress should be stored.
// The progress data includes the total saved leads and credit information.
func SaveProgress(b bool) QueueOpt {
	return func(q *Queue) {
		q.saveProgress = b
	}
}

// Timeout sets the maximum amount of time (in seconds) that can be spent on a chrome automation action.
func Timeout(t time.Duration) QueueOpt {
	return func(q *Queue) {
		q.timeout = t
	}
}

// VPN specifies that the Queue run with OpenVPN.
func VPN(o *openvpn.OpenVPN) QueueOpt {
	return func(q *Queue) {
		q.vpn = o
	}
}

func initAccounts(accounts *[]*models.ApolloAccount, vpn *openvpn.OpenVPN) {
	for _, account := range *accounts {
		if vpn != nil && account.VpnConfig != "" {
			vpn.UpdateUsed(account.VpnConfig)
		}
	}

	for _, account := range *accounts {
		if vpn != nil {
			if account.VpnConfig == "" {
				for _, config := range vpn.Configs {
					if !vpn.ConfigIsUsed(config) {
						account.VpnConfig = config
					}
				}
			}
		}
	}
}

// New instantiates a new Queue instance with the given apollo.io accounts as well as QueueOpts.
func New(accounts []*models.ApolloAccount, opts ...QueueOpt) *Queue {
	q := &Queue{
		limit:     500,
		timeout:   30 * time.Second,
		outputDir: "./apollo-output",
	}

	for _, optFn := range opts {
		optFn(q)
	}

	initAccounts(&accounts, q.vpn)

	jobs := make([]*job, len(accounts))
	for i, account := range accounts {
		jobs[i] = &job{output: strings.ReplaceAll(account.Email, "@", "_"), account: account}
	}

	q.jobs = jobs

	_ = os.MkdirAll(q.outputDir, 0755)

	return q
}

// newChromeContext creates a new chromedp context with the specified configuration.
func (q *Queue) newChromeContext() (context.Context, context.CancelFunc) {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", q.headless),
		chromedp.WindowSize(1920, 1080),
	)
	alloc, _ := chromedp.NewExecAllocator(context.Background(), opts...)

	return chromedp.NewContext(alloc)
}

func (q *Queue) front() (*job, error) {
	if len(q.jobs) == 0 {
		return nil, ErrorEmptyQueue
	}

	return q.jobs[0], nil
}

func (q *Queue) enqueueJob(job *job) {
	q.jobs = append(q.jobs, job)
}

func (q *Queue) dequeueJob() (*job, error) {
	if len(q.jobs) == 0 {
		return nil, ErrorEmptyQueue
	}

	end := len(q.jobs) - 1
	last := q.jobs[end]
	q.jobs = q.jobs[:end]

	return last, nil
}

func (q *Queue) sendToBack() error {
	job, err := q.dequeueJob()
	if err != nil {
		return err
	}
	q.enqueueJob(job)

	return nil
}

func (q *Queue) getOutfileName(ext string) string {
	return filepath.Join(q.outputDir, "leads-"+""+ext)
}
