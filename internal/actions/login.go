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
	"errors"
	"time"

	"github.com/devsheke/scrapollo/internal/models"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/rs/zerolog/log"
)

func isLoggedIn(
	page *rod.Page,
	acc *models.Account,
	timeout time.Duration,
) (bool, error) {
	requiredCookies := make(map[string]*proto.NetworkCookie)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Debug().Str("account", acc.Email).Msg("querying session cookies")
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()

		case <-time.After(500 * time.Millisecond):
		}

		cookies, err := page.Cookies([]string{"https://app.apollo.io"})
		if err != nil {
			return false, err
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
			acc.SetLoginCookies(cookies)
			return true, nil
		}
	}
}

// ApolloLogin is a page action that logs into apollo.io with the specified
// [*models.Account]'s credentials.
func ApolloLogin(browser *rod.Browser, acc *models.Account) (page *rod.Page, err error) {
	page, err = stealth.Page(browser)
	if err != nil {
		return
	}

	log.Info().Str("account", acc.Email).Msg("logging in")

	ok, err := isLoggedIn(page, acc, 60*time.Second)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return
	} else if ok {
		return
	}

	if cookies, ok := acc.GetLoginCookies(); ok && acc.CheckCookieValidity() {
		log.Info().Str("account", acc.Email).Msg("logged in with previously used cookies")
		if err = page.SetCookies(proto.CookiesToParams(cookies)); err != nil {
			return
		}
	}

	err = rod.Try(func() {
		page := page.Timeout(30 * time.Second)
		page.MustNavigate("https://app.apollo.io/#/login").MustWaitDOMStable()
		page.MustElement("input[name=email]").MustInput(acc.Email)
		page.MustElement("input[name=password]").MustInput(acc.Password)
		page.MustElement("button[data-cy=login-button]").MustClick()
	})

	if err != nil {
		return page, err
	}

	ok, err = isLoggedIn(page, acc, 60*time.Second)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return
	} else if ok {
		log.Info().Str("acc", acc.Email).Msg("logged in successfully")
		return
	}

	return page, errors.New("failed to login due to unknown circumstances")
}
