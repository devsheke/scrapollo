package actions

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

// LoginToApollo signs in to Apollo with the agent.
func LoginToApollo(
	b *rod.Browser,
	scraper *models.ApolloAccount,
) (*rod.Page, error) {
	page, err := stealth.Page(b)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("logging into account: %s", scraper.Email)

	if cookies, ok := scraper.GetLoginCookies(); ok && scraper.AreCookiesValid() {
		log.Info().Msg("found pre-existing valid cookies. logged in to apollo.io")
		return page, b.SetCookies(proto.CookiesToParams(cookies))
	}

	err = rod.Try(
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			page := page.Context(ctx)
			page.MustNavigate("https://app.apollo.io/#/login").MustWaitDOMStable()
			page.MustElement("input[name=email]").MustInput(scraper.Email)
			page.MustElement("input[name=password]").MustInput(scraper.Password)
			page.MustElement("button[data-cy=login-button]").MustClick()
		},
	)

	if err != nil {
		return page, err
	}

	requiredCookies := make(map[string]*proto.NetworkCookie)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Debug().Msg("querying cookies to detect auth state")
	for {
		select {
		case <-ctx.Done():
			return page, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		cookies, err := page.Cookies([]string{"https://app.apollo.io"})
		if err != nil {
			return page, err
		}

		for _, cookie := range cookies {
			switch name := cookie.Name; name {
			case "intercom-device-id-dyws6i9m",
				"intercom-session-dyws6i9m",
				"remember_token_leadgenie_v2":
				requiredCookies[name] = cookie
			}
		}

		if len(requiredCookies) == 3 {
			scraper.SetLoginCookies(cookies)
			log.Info().Msg("logged in to apollo.io")
			return page, nil
		}
	}
}
