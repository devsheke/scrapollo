package actions

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

// ErrorListEnd indicates that there are no more new leads to scrape with
// the current set of filters.
var ErrorListEnd = errors.New("reached end of list with given filters")

// Tab constants represent apollo tabs.
const (
	TotalTab  string = "Total"
	NetNewTab string = "Net New"
	SavedTab  string = "Saved"
)

// SelectTab selects the specified Apollo tab on the 'People' page. This function
// assumes the agent is on the 'People' page already and returns ErrorNotOnPeoplePage if
// that is not the case.
func SelectTab(page *rod.Page, tab string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("tab select error: %s", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	log.Info().Msgf("selecting tab: %s", tab)

	err = rod.Try(func() {
		page := page.Context(ctx)
		page.MustElementR(".zp_PfDqP", fmt.Sprintf(`/%s/`, tab)).MustWaitVisible().MustClick()
	})

	return
}

// randomSleep sleeps for a *random amount of time betwen 822ms to 2012ms.
func randomSleep() {
	lower := rand.IntN(1275-822) + 822
	upper := rand.IntN(2012-1457) + 1457
	sleep := rand.IntN(upper-lower) + lower
	log.Debug().Dur("duration ms", time.Duration(sleep)*time.Millisecond).Msg("sleeping")
	time.Sleep(time.Duration(sleep) * time.Millisecond)
}

// SaveLeadsToList saves leads from the 'People' tab on Apollo to a specified list. This function
// assumes that the agent is already on the 'People' page and returns ErrorNotOnPeoplePage if that
// is not the case.
func SaveLeadsToList(
	page *rod.Page,
	listName string,
	timeout time.Duration,
) (int, error) {
	pageInfo, err := GetPageInfo(page)
	if err != nil {
		return 0, err
	}

	if pageInfo.IsLastPage {
		return 0, ErrorListEnd
	}

	err = rod.Try(func() {
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

	return pageInfo.Size, err
}

//go:embed scripts/scrape.js
var scrapeScript string

// ScrapeLeads scrapes all available leads on the given 'People' page on Apollo for the
// given URL or set of filters. This function assumes that the agent is already on the
// 'People' page and returns ErrorNotOnPeoplePage if that is not the case.
func ScrapeLeads(page *rod.Page, tab string, timeout time.Duration) ([]*models.ApolloLead, error) {
	err := rod.Try(func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		page.Context(ctx).MustElement(".zp_tFLCQ .zp_hWv1I").MustWaitVisible()
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var leads []*models.ApolloLead
	obj, err := page.Context(ctx).Eval(scrapeScript)
	if err != nil {
		return nil, err
	}

	err = obj.Value.Unmarshal(&leads)
	if err != nil {
		return nil, err
	}
	return leads, nil
}
