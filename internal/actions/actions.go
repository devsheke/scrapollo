package actions

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog/log"
)

// ErrorNotOnPeoplePage indicates that the agent is not on the 'People' page of apollo.io.
var ErrorNotOnPeoplePage = errors.New("cannot go to list as agent is not on the 'People' page")

type dialog struct {
	class, regex string
}

var (
	newUIDialog   = dialog{".zp_p2Xqs.zp_v565m.zp_qhNxC", "Skip tour"}
	powerUpDialog = dialog{".zp_tZMYK", "Got it"}
)

// closeDialog closes the Apollo dialog which prompts users to try out their new UI.
func closeDialog(page *rod.Page, dialog dialog) error {
	log.Debug().Msg("checking if any annoying dialogs are present")
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		el, err := page.Context(ctx).ElementR(dialog.class, dialog.regex)
		switch err {
		case context.DeadlineExceeded, &rod.ElementNotFoundError{}:
			return nil
		default:
			if err != nil {
				return err
			}
		}

		if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return err
		}

		log.Info().Msg("removed popup")
		time.Sleep(5 * time.Second)
	}
}

func ClosePowerUpDialog(page *rod.Page, timeout time.Duration) error {
	return closeDialog(page, powerUpDialog)
}

func CloseNewUIDialog(page *rod.Page, timeout time.Duration) error {
	return closeDialog(page, newUIDialog)
}

// GoToList navigates the agent to the specified Apollo lead list. This function returns
// ErrorNotOnPeoplePage if the agent not currently on the "People" page of apollo.io.
func GoToList(page *rod.Page, listName string, timeout time.Duration) error {
	openState := ".zp_YkfVU"
	filterAccordion := ".zp-accordion.zp_UeG9f.zp_p8DhX"
	accordianToggle := ".zp-accordion-header.zp_r3aQ1"

	err := rod.Try(func() {
		if !strings.HasPrefix(page.MustInfo().URL, "https://app.apollo.io/#/people") {
			panic(ErrorNotOnPeoplePage)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		page := page.Context(ctx)
		log.Debug().Msg("searching for accordians")
		page.MustElement(filterAccordion).
			MustWaitVisible()
		accordians := page.MustElements(filterAccordion)

		if len(accordians) < 11 {
			panic(fmt.Errorf("unexpected number of filter accordians: %d", len(accordians)))
		}

		listAccordian := accordians[0]
		class := listAccordian.MustAttribute("class")

		if !strings.Contains(*class, openState) {
			listAccordian.MustElement(accordianToggle).MustClick()
		}

		listAccordian.MustElement(".Select-input").MustInput(listName)
		page.Keyboard.MustType(input.Enter)
	})

	return err
}

// PageInfo represents the page data of the current page of leads
// for the given filters on apollo.
type PageInfo struct {
	PageNumber, Start, End, Size int
	IsLastPage                   bool
	PageButtons                  rod.Elements
}

// GetPageInfo returns an instance of *PageInfo based on the current page.
func GetPageInfo(page *rod.Page) (*PageInfo, error) {
	p := new(PageInfo)
	var err error

	log.Debug().Msg("getting page size info")
	err = rod.Try(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		pageInfo := page.Context(ctx).MustElement(".zp_xAPpZ").MustWaitVisible().MustText()

		infoSplit := strings.Split(pageInfo, " ")
		if p.Start, err = strconv.Atoi(infoSplit[0]); err != nil {
			panic(err)
		}
		if p.End, err = strconv.Atoi(infoSplit[2]); err != nil {
			panic(err)
		}
		p.Size = p.End - p.Start + 1
	})

	if errors.Is(err, context.DeadlineExceeded) {
		err := rod.Try(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			page.Context(ctx).
				MustElementR(".zp_MVq1c", "No people match your criteria").
				MustWaitVisible()
		})

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}

		return nil, ErrorListEnd
	}

	if err != nil {
		return nil, err
	}

	log.Debug().Msg("getting page status")
	err = rod.Try(func() {
		p.PageNumber, err = strconv.Atoi(page.MustElement(".zp_jzp8p").MustWaitVisible().MustText())
		if err != nil {
			panic(err)
		}

		pageBtns := page.MustElements(".zp_m_JQ3 > .zp_qe0Li.zp_S5tZC")
		if len(pageBtns) < 2 {
			panic(fmt.Errorf("not enough page buttons found"))
		}

		if attr, err := pageBtns.Last().Attribute("disabled"); err != nil {
			panic(err)
		} else {
			p.IsLastPage = attr != nil
		}
		p.PageButtons = pageBtns
	})

	return p, err
}

// GoToNextPage navigates the agent to the next page of 'People' or apollo leads
// for the given URL or set of filters.
//
// It returns ErrorListEnd if there are no more pages available.
func GoToNextPage(page *rod.Page) error {
	log.Debug().Msg("going to next page")

	info, err := GetPageInfo(page)
	if err != nil {
		return err
	}

	if info.IsLastPage {
		log.Debug().Msg("reached last page")
		return ErrorListEnd
	}

	err = info.PageButtons.Last().Click(proto.InputMouseButtonLeft, 1)
	return err
}

// TODO
func GoToPage(page *rod.Page, pageNumber int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := rod.Try(func() {
		page = page.Context(ctx)
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
