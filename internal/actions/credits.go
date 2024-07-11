package actions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

// FetchCredits fetches the credit usage data for the logged in apollo user.
//
// # NOTE: a user must be logged into apollo for this to work.
func FetchCredits(ctx context.Context, timeout time.Duration) (int, time.Time, error) {
	log.Info().Msg("now fetching credits")

	_ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Debug().Msg("navigating to credits page")

	var t time.Time
	err := chromedp.Run(_ctx, chromedp.Navigate("https://app.apollo.io/#/settings/credits/current"))
	if err != nil {
		return 0, t, err
	}

	var renewal, credits string

	_ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	err = chromedp.Run(
		_ctx,
		chromedp.Text(".zp_SJzex", &renewal, chromedp.ByQuery, chromedp.NodeVisible),
		chromedp.EvaluateAsDevTools(
			`document.querySelectorAll(".zp_ajv0U")[1].innerText`,
			&credits,
		),
	)
	if err != nil {
		return 0, t, err
	}

	log.Debug().Str("credits", credits).Str("renewal", renewal).Msg("fetched credit data")

	renewalSplit := strings.Split(renewal, ": ")
	if len(renewalSplit) < 2 {
		return 0, t, fmt.Errorf("incorrect t string: %q", renewal)
	}

	zone, _ := time.Now().Zone()
	t, err = time.Parse(models.TimeFormat, fmt.Sprintf("%s %s", renewalSplit[1], zone))
	if err != nil {
		return 0, t, err
	}

	creditsSplit := strings.Split(credits, " ")
	if len(creditsSplit) < 6 {
		return 0, t, fmt.Errorf("invalid credit usage string: %q", credits)
	}

	used, err := strconv.Atoi(strings.ReplaceAll(creditsSplit[0], ",", ""))
	if err != nil {
		return 0, t, err
	}

	total, err := strconv.Atoi(strings.ReplaceAll(creditsSplit[2], ",", ""))
	if err != nil {
		return 0, t, err
	}

	return total - used, t, nil
}
