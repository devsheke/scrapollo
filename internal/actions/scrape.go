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

package actions

import (
	"context"
	_ "embed"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/devsheke/scrapollo/internal/models"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/rs/zerolog/log"
)

type ApolloTab string

const (
	TotalTab  ApolloTab = "Total"
	NetNewTab ApolloTab = "Net New"
	SavedTab  ApolloTab = "Saved"
)

func (tab ApolloTab) Select(page *rod.Page) (err error) {
	log.Debug().Str("tab", string(tab)).Msg("selecting tab")

	defer func() {
		if err != nil {
			err = fmt.Errorf("tab select error: %s", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err = rod.Try(func() {
		page := page.Context(ctx)
		page.MustElementR(".zp_PfDqP", fmt.Sprintf(`/%s/`, tab)).MustWaitVisible().MustClick()
	})

	return
}

// randomSleep sleeps for a *random amount of time betwen 822ms to 2476ms.
func randomSleep() {
	lower := rand.IntN(1275-822) + 822
	upper := rand.IntN(2476-1457) + 1457
	sleep := rand.IntN(upper-lower) + lower
	time.Sleep(time.Duration(sleep) * time.Millisecond)
}

func SaveLeads(page *rod.Page, listName string, timeout time.Duration) error {
	log.Info().Str("list", listName).Msg("saving leads")
	err := rod.Try(func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		page := page.Context(ctx)
		page.MustElement(".zp_wMhzv").MustWaitVisible().MustClick()
		page.MustElement("button[type=submit].zp_qe0Li.zp_FG3Vz.zp_rsjqe.zp_h2EIO").
			MustWaitVisible().
			MustClick()

		page.MustElement("button.zp_qe0Li.zp_FG3Vz.zp_rsjqe.zp_h2EIO").
			MustWaitVisible().
			MustClick()

		page.MustElement(".zp-modal-content.zp_AX8K7.zp_qTumF.zp_esFCS").
			MustWaitVisible().
			MustElement(".Select-input").
			MustInput(listName)

		for range 2 {
			page.Keyboard.MustType(input.Enter)
			randomSleep()
		}

		page.MustElement(".zp_VfG2H.zp_cUvBN").MustWaitVisible()
		page.MustReload()
	})

	return err
}

//go:embed scripts/scrape.js
var scrapeScript string

func ScrapeLeads(page *rod.Page, timeout time.Duration) ([]*models.Lead, error) {
	log.Debug().Msg("scraping leads")

	err := rod.Try(func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		page.Context(ctx).MustElement(".zp_tFLCQ .zp_hWv1I").MustWaitVisible()
	})

	if err != nil {
		return nil, err
	}

	var leads []*models.Lead

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Debug().Msg("running scrape script")
	result, err := page.Context(ctx).Eval(scrapeScript)
	if err != nil {
		return nil, err
	}

	log.Debug().Msg("unmarshaling scraped values")
	err = result.Value.Unmarshal(&leads)
	return leads, err
}