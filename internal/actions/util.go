package actions

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
)

var (
	// ErrorNoNewLeads indicates that there are no more leads to scrape for the given filter.
	ErrorNoNewLeads = errors.New("no more new leads to scrape")
	// ErrorNotAllowed indicates that the current scraper is not allowed to scrape any more
	// leads due to some restriction.
	ErrorNotAllowed = errors.New("cannot continue scraping due to a restriction")
)

// PollTableData polls for the table's 'dataset.cyLoaded' to equal 'true'.
// The aforementioned condition indicates that all leads for the current page have been loaded.
func PollTableData(ctx context.Context, timeout time.Duration) error {
	log.Debug().Msg("polling for data")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fn := `() => {
      try {
        return document.querySelector(".zp_G5KZB").dataset.cyLoaded === "true"
      } catch(e) {
        return false
      }
  }`

	err := chromedp.Run(ctx, chromedp.PollFunction(fn, nil))
	if err != nil {
		return err
	}

	log.Debug().Msg("table data loaded")

	return nil
}

// GetPageData gets the current page data which includes:
//
// *  curr  - number of records on current page.
//
// *  limit - total number of records for the given filters.
func GetPageData(ctx context.Context) (curr int, limit int, err error) {
	log.Debug().Msg("getting page count")

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var pageData string
	err = chromedp.Run(ctx, chromedp.Text("span.zp_VVYZh", &pageData, chromedp.ByQuery))
	if err != nil {
		return
	}

	if pageData == "" {
		log.Warn().Msg("page count error empty")
		err = ErrorNoNewLeads
		return
	}

	log.Debug().Str("info", pageData).Msg("got page count data")

	pageDataSplit := strings.Split(strings.ReplaceAll(pageData, ",", ""), " ")
	if curr, err = strconv.Atoi(pageDataSplit[2]); err != nil {
		return
	}

	limit, err = strconv.Atoi(pageDataSplit[len(pageDataSplit)-1])

	return
}

// TakeScreenshot takes a screenshot of the given Chrome screen and saves it to the
// specified path.
func TakeScreenshot(ctx context.Context, path string) error {
	log.Debug().Msg("taking error screenshot")

	var b []byte
	err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&b))
	if err != nil {
		return err
	}

	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return err
	}

	log.Debug().Str("path", path).Msg("creating image file for screenshot")

	imgfile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer imgfile.Close()

	log.Debug().Str("path", path).Msg("saved screenshot")

	return png.Encode(imgfile, img)
}

func HideEmailBox(ctx context.Context) error {
	log.Debug().Msg("adding stylesheet to hide email popovers")

	fn := `
  (css) => {
    const style = document.createElement('style');
    style.type = 'text/css';
    style.appendChild(document.createTextNode(css));
    document.head.appendChild(style);
    return true;
  }
  `
	return chromedp.Run(ctx, chromedp.PollFunction(
		fn, nil, chromedp.WithPollingArgs(".zp_YI5xm { display: none; }"),
	))
}

func CheckAllowed(ctx context.Context) error {
	log.Debug().Msg("checking if process can continue")

	var isAllowed bool
	err := chromedp.Run(ctx,
		chromedp.Sleep(2*time.Second),
		chromedp.EvaluateAsDevTools(
			`document.getElementsByClassName("zp_RB9tu zp_HGOPM").length <= 0`,
			&isAllowed,
		))
	if err != nil {
		return err
	}

	if !isAllowed {
		return ErrorNotAllowed
	}

	return nil
}

// NextPage clicks on the next page button if it exists or is enabled. It returns ErrorListEnd
// if the next page button is disabled.
func NextPage(ctx context.Context, timeout time.Duration) error {
	var bottom []*cdp.Node

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.Nodes(".zp_mYjg6", &bottom, chromedp.ByQuery)); err != nil {
		return err
	} else if len(bottom) < 1 {
		return errors.New("bottom bar not found")
	}

	var buttons []*cdp.Node
	err := chromedp.Run(
		ctx,
		chromedp.Nodes(
			"button[aria-label=right-arrow].zp-button.zp_zUY3r.zp_MCSwB.zp_xCVC8",
			&buttons,
			chromedp.FromNode(bottom[0]),
			chromedp.ByQuery,
		),
	)
	if err != nil {
		return err
	}

	if len(buttons) < 1 {
		return errors.New("could not find next page button")
	}

	nextButton := buttons[len(buttons)-1]
	if _, disabled := nextButton.Attribute("disabled"); disabled {
		return ErrorListEnd
	}

	return chromedp.Run(ctx, chromedp.Click(nextButton.FullXPathByID()))
}

func goToPage(ctx context.Context, page string) error {
	var bottom []*cdp.Node

	if err := chromedp.Run(ctx, chromedp.Nodes(".zp_mYjg6", &bottom, chromedp.ByQuery)); err != nil {
		return err
	} else if len(bottom) < 1 {
		return errors.New("bottom bar not found")
	}

	err := chromedp.Run(
		ctx,
		chromedp.Click("div[role=combobox]", chromedp.FromNode(bottom[0]), chromedp.ByQuery),
	)
	if err != nil {
		return nil
	}

	var pageBtns []*cdp.Node
	err = chromedp.Run(ctx, chromedp.Nodes(".Select-option", &pageBtns, chromedp.ByQueryAll))
	if err != nil {
		return err
	}

	i := slices.IndexFunc(pageBtns, func(c *cdp.Node) bool {
		var text string
		if err := chromedp.Run(ctx, chromedp.Text(c.FullXPathByID(), &text)); err != nil {
			return false
		}

		return strings.TrimSpace(text) == page
	})

	if i < 0 {
		return errors.New("page not found")
	}

	return chromedp.Run(ctx, chromedp.Click(pageBtns[i].FullXPathByID()))
}
