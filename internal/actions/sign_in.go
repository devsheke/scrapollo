package actions

import (
	"context"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
)

func signInTasks(email, password string) chromedp.Tasks {
	emailInput := `//input[@name="email"]`

	return chromedp.Tasks{
		chromedp.WaitVisible(emailInput),
		chromedp.SendKeys(emailInput, email),
		chromedp.SendKeys(`//input[@name="password"]`, password),
		chromedp.Click(`//button[@type="submit"]`),
	}
}

// SignIn uses the given credentials to sign into the apollo.io dashboard and polls the
// user's log in status.
func SignIn(ctx context.Context, email, password string, timeout time.Duration) error {
	log.Debug().Str("scraper", email).Msg("signing in to apollo.io")

	err := chromedp.Run(ctx, chromedp.Navigate("https://app.apollo.io/#/login"))
	if err != nil {
		return err
	}

	log.Debug().Msg("entering credentials")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err = chromedp.Run(ctx, signInTasks(email, password)); err != nil {
		return err
	}

	for {
		var url string
		err := chromedp.Run(ctx, chromedp.Location(&url))
		if err != nil {
			return err
		}

		if strings.Contains(url, "onboarding") || strings.Contains(url, "control-center") {
			return nil
		}
	}
}
