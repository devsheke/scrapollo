package actions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

// FetchCreditUsage fetches the credit usage data for the logged in apollo user.
// This function assumes that the agent has already logged into apollo.
func FetchCreditUsage(
	page *rod.Page,
	account *models.ApolloAccount,
	timeout time.Duration,
) (credits int, refreshTime *models.Time, err error) {
	var creditsText []string
	err = rod.Try(func() {
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

	refreshTime = models.NewTime(_time)

	return
}
