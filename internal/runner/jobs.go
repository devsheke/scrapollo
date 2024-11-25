package runner

import (
	"container/list"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

var ErrorEmptyQueue = errors.New("job queue is empty")

// ScrapeJob represents a scrape task. It includes an apollo.io account
// as well as an output destination for the scrape data.
type ScrapeJob struct {
	account    *models.ApolloAccount
	startedAt  *models.Time
	outputName string
	saved      int
}

func (j *ScrapeJob) UpdateStart() {
	j.startedAt.Set(time.Now())
}

func (j *ScrapeJob) IsDoneForToday(dailyLimit int) bool {
	if !j.startedAt.IsSome() {
		return false
	}

	cond := time.Now().Before(j.startedAt.Get())
	if cond && j.saved > dailyLimit {
		j.Reset()
		return true
	}

	return false
}

func (j *ScrapeJob) Reset() {
	j.startedAt.Reset()
	j.saved = 0
}

type JobQueue struct {
	*list.List
}

func (j *JobQueue) IsEmpty() bool {
	return j.Len() == 0
}

func (j *JobQueue) Requeue() error {
	if j.IsEmpty() {
		return ErrorEmptyQueue
	}

	j.MoveToBack(j.Front())
	return nil
}

func (j *JobQueue) Iter(yield func(*ScrapeJob) bool) {
	if j.IsEmpty() {
		return
	}

	for item := j.Front(); item != nil; item = item.Next() {
		job, _ := item.Value.(*ScrapeJob)
		if !yield(job) {
			return
		}
	}
}

func NewJobQueue(accounts []*models.ApolloAccount) *JobQueue {
	q := list.New()

	for _, account := range accounts {
		if account.SaveToList == "" {
			account.SaveToList = uuid.NewString() + "-" + time.Now().Format(time.RFC3339)
		}
		job := &ScrapeJob{
			account:    account,
			startedAt:  &models.Time{},
			outputName: strings.ReplaceAll(account.Email, "@", "_"),
		}
		q.PushBack(job)
	}

	return &JobQueue{q}
}
