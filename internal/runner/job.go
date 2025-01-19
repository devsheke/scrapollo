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
	"iter"
	"strings"
	"time"

	"github.com/devsheke/scrapollo/internal/models"
)

type job struct {
	acc        *models.Account
	savedToday int
	startedAt  *models.Time
}

func (j *job) hitDailyLimit(limit int) bool {
	startedAt, ok := j.startedAt.Get()
	if !ok {
		return false
	}

	cond := time.Now().Before(startedAt.Add(24 * time.Hour))
	if cond && j.savedToday >= limit {
		j.reset()
		return true
	}

	return false
}

func (j *job) incrementSaved(amount int) {
	j.acc.Increment(amount)
	j.acc.UseCredits(amount)
	j.savedToday += amount
}

func (j *job) reset() {
	j.savedToday = 0
	j.startedAt.Reset()
}

func (j *job) start() {
	j.startedAt = models.NewTimeValid(time.Now())
}

type queue struct {
	*list.List
}

func newQueue(accs []*models.Account) *queue {
	q := list.New()
	for _, acc := range accs {
		job := &job{
			acc:       acc,
			startedAt: models.NewTime(),
		}

		if acc.List == "" {
			acc.List = "scrapollo-run-" + strings.ReplaceAll(acc.Email, "@", "_")
		}

		q.PushBack(job)
	}

	return &queue{q}
}

func (q *queue) isEmpty() bool {
	return q.Len() == 0
}

func (q *queue) iter() iter.Seq2[int, *job] {
	return func(yield func(int, *job) bool) {
		if q.isEmpty() {
			return
		}

		for idx, item := 0, q.Front(); item != nil; idx, item = idx+1, item.Next() {
			job, _ := item.Value.(*job)
			if !yield(idx, job) {
				return
			}
		}
	}
}

func (q *queue) requeue() error {
	if q.isEmpty() {
		return errors.New("failed to requeue job in an empty queue")
	}
	q.MoveToBack(q.Front())

	return nil
}
