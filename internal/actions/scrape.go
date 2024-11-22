package actions

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
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
	tab string,
	listName string,
	timeout time.Duration,
) (int, error) {
	if err := closeNewUIDialog(page); err != nil {
		return 0, err
	}

	if err := SelectTab(page, tab); err != nil {
		return 0, err
	}

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

// ScrapeLeads scrapes all available leads on the given 'People' page on Apollo for the
// given URL or set of filters. This function assumes that the agent is already on the
// 'People' page and returns ErrorNotOnPeoplePage if that is not the case.
func ScrapeLeads(page *rod.Page, tab string) ([]*models.ApolloLead, error) {
	if err := closeNewUIDialog(page); err != nil {
		return nil, err
	}

	var rows []*rod.Element
	err := rod.Try(func() {
		page.MustElement(".zp_hWv1I").MustWaitVisible()
		rows = page.MustElements(".zp_hWv1I")
	})
	if err != nil {
		return nil, err
	}

	leads := make([]*models.ApolloLead, len(rows)-1)
	for i, row := range rows[1:] {
		columns, err := row.Elements(".zp_KtrQp")
		if err != nil {
			return nil, err
		}

		lead := new(models.ApolloLead)
		err = rod.Try(func() {
			lead.Name = strings.ReplaceAll(columns[1].MustText(), "\n------", "")
			lead.Title = columns[2].MustText()
			lead.Company = columns[3].MustText()
			lead.Location = columns[8].MustText()
			lead.Employees = columns[9].MustText()
			lead.Industry = columns[10].MustText()
			lead.Keywords = strings.ReplaceAll(columns[11].MustText(), "\n", ",")
		})
		if err != nil {
			return nil, err
		}

		linkedinCol, err := columns[7].Element("a")
		if err == nil {
			err = rod.Try(func() {
				lead.LinkedIn = *linkedinCol.MustAttribute("href")
			})
			if err != nil {
				return nil, err
			}
		}

		err = rod.Try(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			for {
				select {
				case <-ctx.Done():
					panic(ctx.Err())
				case <-time.After(500 * time.Millisecond):
				}
				emailSpan, err := columns[4].Element(".zp_xvo3G")
				if err != nil {
					if errors.Is(err, &rod.ElementNotFoundError{}) {
						continue
					}
					panic(err)
				}

				lead.Email = []string{emailSpan.MustText()}
				break
			}

			lead.Phone, err = columns[5].Text()
			if err != nil {
				log.Error().Err(err).Msgf("failed to get phone for: %q", lead.Name)
				return
			}
		})

		log.Info().Msgf("%+v", lead)

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}

		leads[i] = lead
	}

	return leads, nil
}
