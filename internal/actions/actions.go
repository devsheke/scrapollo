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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/devsheke/scrapollo/internal/models"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog/log"
)

var (
	// ErrorListEnd is an error which is returned when there are no more leads to save/scrape
	// in the given list, i.e., we have reached the end of the list.
	ErrorListEnd = errors.New("reached end of list")

	// ErrorNavButtonsNotFound is an error returned when the page navigation buttons are not found
	// on the current page.
	ErrorNavButtonsNotFound = errors.New("failed to find page navigation buttons")
)

// PageData represents all available data regarding the page number,
// size, etc., on the current page.
type PageData struct {
	Number, Start, End, Size, TotalSize int
	LastPage                            bool
	NavButtons                          rod.Elements
}

// NextPage is a method for navigating to the next available page for
// the given set of filters.
func (pd *PageData) NextPage(page *rod.Page) error {
	log.Debug().Msg("going to next page")

	if pd.LastPage {
		return ErrorListEnd
	}

	switch btn := pd.NavButtons.Last(); btn {
	case nil:
		return ErrorNavButtonsNotFound

	default:
		return btn.Click(proto.InputMouseButtonLeft, 1)
	}
}

// GetPageData is a page action that returns a [*PageData] value representing all available
// data regarding the page number, size, etc,. This function only works on the 'People' page
// on Apollo. This function assumes you're on the 'People' page on Apollo.
func GetPageData(page *rod.Page, timeout time.Duration) (pd *PageData, err error) {
	log.Debug().Msg("getting page data")

	err = rod.Try(func() {
		log.Debug().Msg("parsing page size information")

		info := strings.Split(
			page.Timeout(timeout).MustElement(".zp_xAPpZ").MustWaitVisible().MustText(),
			" ",
		)

		if pd.Start, err = strconv.Atoi(info[0]); err != nil {
			panic(err)
		}

		if pd.End, err = strconv.Atoi(info[2]); err != nil {
			panic(err)
		}

		if pd.TotalSize, err = strconv.Atoi(strings.ReplaceAll(info[4], ",", "")); err != nil {
			panic(err)
		}

		pd.Size = pd.End - pd.Start + 1
	})

	if errors.Is(err, context.DeadlineExceeded) {
		err := rod.Try(func() {
			page.Timeout(20*time.Second).
				MustElementR(".zp_MVq1c", "No people match your criteria").
				MustWaitVisible()
		})

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}

		return nil, ErrorListEnd
	} else if err != nil {
		return
	}

	err = rod.Try(func() {
		log.Debug().Msg("getting page navigation information")

		numText := page.Timeout(20 * time.Second).
			MustElement(".zp_jzp8p").
			MustWaitVisible().
			MustText()

		if pd.Number, err = strconv.Atoi(numText); err != nil {
			panic(err)
		}

		navBtns := page.MustElements(".zp_m_JQ3 > .zp_qe0Li.zp_S5tZC")
		if len(navBtns) < 2 {
			panic(fmt.Errorf("not enough page buttons found"))
		}

		if attr, err := navBtns.Last().Attribute("disabled"); err != nil {
			panic(err)
		} else {
			pd.LastPage = attr != nil && *attr == "true"
		}

		pd.NavButtons = navBtns
	})

	return pd, err
}

// GoToPage is a page navigation function that navigates to the specified page
// number on the 'People' page on Apollo. This function assumes you're on the 'People'
// page on Apollo.
func GoToPage(page *rod.Page, pageNumber int, timeout time.Duration) error {
	log.Debug().Int("number", pageNumber).Msg("navigating to page")

	page = page.Timeout(timeout)
	err := rod.Try(func() {
		page.MustElement(".zp_VTl3h.zp_xqxgc .zp_dJ2fA").MustWaitVisible()

		inputs := page.MustElements(".zp_VTl3h.zp_xqxgc .zp_dJ2fA")
		if len(inputs) < 2 {
			panic("could not find page control switch")
		}

		inputs[1].MustClick()

		listbox := page.MustElement("[role=listbox]").MustWaitVisible()
		listbox.MustElement("a").MustWaitVisible()

		pages := listbox.MustElements("a")
		if len(pages) < pageNumber-1 {
			panic("found too few page number option")
		}

		pages[pageNumber-1].MustClick()
	})

	return err
}

// GrabErrorSnapshot is a page action which grabs a screenshot and the rendered
// HTML of the current page and saves them in the specified directory.
func GrabErrorSnapshot(page *rod.Page, acc *models.Account, errorDir string) error {
	log.Debug().Str("account", acc.Email).Msg("grabbing error snapshot")

	err := rod.Try(func() {
		ssFile := filepath.Join(errorDir, acc.Email+".png")
		page.MustScreenshot(ssFile)

		htmlFile := filepath.Join(errorDir, acc.Email+".html")
		if err := os.WriteFile(htmlFile, []byte(page.MustHTML()), 0644); err != nil {
			panic(err)
		}
	})

	return err
}

const (
	accordianOpenState     string = ".zp_YkfVU"
	accordianToggleElement string = ".zp-accordion.zp_UeG9f.zp_p8DhX"
	filterAccordionElement string = ".zp-accordion-header.zp_r3aQ1"
	peoplePageURL          string = "https://app.apollo.io/#/people"
)

// LocateList is a page action that navigates to the Apollo list with the provided listName.
func LocateList(page *rod.Page, listName string, timeout time.Duration) error {
	log.Debug().Str("list", listName).Msg("locating list")

	err := rod.Try(func() {
		if !strings.HasPrefix(page.MustInfo().URL, peoplePageURL) {
			page.MustNavigate(peoplePageURL).MustWaitDOMStable()
		}

		page := page.Timeout(timeout)
		page.MustElement(filterAccordionElement).
			MustWaitVisible()

		accordians := page.MustElements(filterAccordionElement)
		if len(accordians) < 11 {
			panic(fmt.Errorf("unexpected number of filter accordians: %d", len(accordians)))
		}

		listAccordian := accordians[0]
		class := listAccordian.MustAttribute("class")

		if !strings.Contains(*class, accordianOpenState) {
			listAccordian.MustElement(accordianToggleElement).MustClick()
		}

		listAccordian.MustElement(".Select-input").MustInput(listName)
		page.Keyboard.MustType(input.Enter)
	})

	return err
}
