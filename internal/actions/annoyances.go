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

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog/log"
)

// Annoyance is represents any sort of annoyance that affects the
// scraping flow.
type Annoyance struct {
	Name, Regex, Selector string
	ActionFunc            func(*rod.Element) error
}

func simpleClick(e *rod.Element) error {
	return e.Click(proto.InputMouseButtonLeft, 1)
}

var (
	// NewUIAnnoyance represents the dialog shown to users using the
	// updated Apollo UI.
	NewUIAnnoyance = &Annoyance{
		Name:       "New UI popup",
		Regex:      "Skip tour",
		Selector:   ".zp_p2Xqs.zp_v565m.zp_qhNxC",
		ActionFunc: simpleClick,
	}

	// PopupDialogAnnoyance represents a generic popup dialog shown at random
	// moments while using Apollo.
	PopupDialogAnnoyance = &Annoyance{
		Name:       "Dialog popups",
		Regex:      "Got it",
		Selector:   ".zp_tZMYK",
		ActionFunc: simpleClick,
	}

	// SidenavAnnoyance represents the Apollo side bar which affects scraping
	// while using the rod stealth library.
	SidenavAnnoyance = &Annoyance{
		Name:       "Sidebar",
		Selector:   "#side-nav",
		ActionFunc: func(e *rod.Element) error { return e.Remove() },
	}

	// TopBannerAnnoyance represents the banner at the top of the Apollo page which
	// sometimes appears and hinders scraping.
	TopBannerAnnoyance = &Annoyance{
		Name:       "Top banner",
		Selector:   "[data-variant=black] button[aria-label=Dismiss]",
		ActionFunc: simpleClick,
	}
)

// RemoveAnnoyanceis a page action which searches for all available instances of the specified
// [*Annoyance] on the current page and performs the action specified by [*Annoyance.ActionFunc]
// for each of them.
func RemoveAnnoyance(page *rod.Page, annoyance *Annoyance, timeout time.Duration) error {
	log.Debug().Str("annoyance", annoyance.Name).Msg("attempting to remove annoyance")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		var element *rod.Element
		err := rod.Try(func() {
			page = page.Context(ctx)
			if annoyance.Regex != "" {
				element = page.MustElementR(annoyance.Selector, annoyance.Regex).MustWaitVisible()
			} else {
				element = page.MustElement(annoyance.Selector).MustWaitVisible()
			}
		})

		if errors.Is(err, context.DeadlineExceeded) {
			log.Debug().Str("annoyance", annoyance.Name).Msg("annoyance not found")
			return nil
		} else if err != nil {
			return err
		}

		if err := annoyance.ActionFunc(element); err != nil {
			return err
		}

		log.Debug().Str("annoyance", annoyance.Name).Msg("removed annoyance")

		time.Sleep(2 * time.Second)
	}
}
