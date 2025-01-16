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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/devsheke/scrapollo/internal/models"
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// FetchCreditUsage is a page action that fetches credit usage information for the provided
// [*models.Account] from Apollo. This function returns the amount of credits remaining and
// a [*models.Time] value indicating when credits will be renewed.
func FetchCreditUsage(
	page *rod.Page,
	acc *models.Account,
	timeout time.Duration,
) (credits int, refreshTime *models.Time, err error) {
	log.Info().Str("account", acc.Email).Msg("fetching credit usage")

	var creditsText []string
	err = rod.Try(func() {
		log.Info().Msg("fetching credit data")

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		page := page.Context(ctx)
		creditElem := ".zp_ZlMia"
		page.MustNavigate("https://app.apollo.io/#/settings/credits/current").
			MustElement(creditElem).
			MustWaitVisible()

		elems := page.MustElements(creditElem)
		if len(elems) != 4 {
			panic("unexpected number of credit elements found")
		}

		creditsText = strings.Split(elems[1].MustText(), " ")
		if len(creditsText) != 6 {
			panic(fmt.Sprintf("unexpected credit string found: %q", strings.Join(creditsText, " ")))
		}
	})

	if err != nil {
		return
	}

	creditsUsed, err := strconv.Atoi(strings.ReplaceAll(creditsText[0], ",", ""))
	if err != nil {
		err = fmt.Errorf("failed to fetch used credits amnt: %s", err)
		return
	}

	creditsMax, err := strconv.Atoi(strings.ReplaceAll(creditsText[2], ",", ""))
	if err != nil {
		err = fmt.Errorf("failed to fetch max credits amnt: %s", err)
		return
	}

	credits = creditsMax - creditsUsed

	var creditsRenewal string
	err = rod.Try(func() {
		log.Info().Msg("fetching renewal data")

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		page := page.Context(ctx)
		if text := page.MustElement(".zp_jtf9O").MustWaitVisible().MustText(); len(text) < 30 {
			panic(fmt.Errorf("unexpected credit renewal string: %q", text))
		} else {
			creditsRenewal = text[29:]
		}
	})

	if err != nil {
		return
	}

	_time, err := time.Parse(models.TimeFormat, creditsRenewal)
	if err != nil {
		return
	}

	refreshTime = models.NewTimeValid(_time)
	return
}
