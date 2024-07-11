package actions

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
)

// These constants represents the tab names available on
// the people finder page on apollo.io.
const (
	TotalTab  string = "Total"
	NetNewTab string = "Net New"
	SavedTab  string = "Saved"
)

// SetTab selects the specified tab on the people finder page on apollo.io.
// # NOTE: this function requires you to be logged into apollo.
func SetTab(ctx context.Context, tab string, timeout time.Duration) error {
	log.Info().Str("tab", tab).Msg("setting tab")

	tabsPollFn := `
    () => document.querySelectorAll(".zp-link.zp_OotKe.zp_LdIJ3.zp_n5qRT").length >= 3
  `

	var tabs []*cdp.Node
	err := chromedp.Run(
		ctx,
		chromedp.PollFunction(tabsPollFn, nil, chromedp.WithPollingTimeout(timeout)),
		chromedp.Nodes(
			".zp-link.zp_OotKe.zp_LdIJ3.zp_n5qRT",
			&tabs,
			chromedp.NodeVisible,
			chromedp.ByQueryAll,
		),
	)
	if err != nil {
		return err
	}

	log.Debug().Msgf("got: %d tabs", len(tabs))

	tabPos := slices.IndexFunc(tabs, func(t *cdp.Node) bool {
		var text string
		if err := chromedp.Run(ctx, chromedp.Text(t.FullXPathByID(), &text)); err != nil {
			return false
		}

		log.Debug().Msgf("found tab: %q", text)
		return strings.Contains(text, tab)
	})

	if tabPos < 0 {
		return fmt.Errorf("failed to find tab: %q", tab)
	}

	if err := chromedp.Run(ctx, chromedp.Click(tabs[tabPos].FullXPathByID())); err != nil {
		return err
	}

	if err := PollTableData(ctx, timeout); err != nil {
		return err
	}

	log.Debug().Str("tab", tab).Msg("located tab")

	return nil
}
