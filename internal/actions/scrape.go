package actions

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

var (
	// ErrorNoEmail indicates that there were no emails to scrape for the current lead.
	ErrorNoEmail = errors.New("email button not found")
	// ErrorListEnd indicates that the end of the given list of leads has been reached.
	ErrorListEnd = errors.New("no more leads to scrape")
)

func getInnerText(ctx context.Context, node *cdp.Node) (text string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	err = chromedp.Run(ctx, chromedp.Text(node.FullXPathByID(), &text))
	if errors.Is(err, context.DeadlineExceeded) {
		err = nil
	}

	return
}

func getLinks(ctx context.Context, fromNode *cdp.Node) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	var nodes []*cdp.Node
	err := chromedp.Run(
		ctx,
		chromedp.Nodes(
			".zp-link.zp_OotKe",
			&nodes,
			chromedp.FromNode(fromNode),
			chromedp.ByQueryAll,
		),
	)
	if err != nil {
		return nil, err
	}

	if len(nodes) < 1 {
		return nil, nil
	}

	links := make([]string, 0, len(nodes))
	for _, node := range nodes {
		links = append(links, node.AttributeValue("href"))
	}

	return links, nil
}

func getEmails(ctx context.Context, fromNode *cdp.Node) ([]string, error) {
	// Look for 'Access email' button or the email pop-up button.
	_ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var buttons []*cdp.Node
	err := chromedp.Run(
		_ctx,
		chromedp.Nodes(
			".zp-button.zp_zUY3r.zp_n9QPr.zp_MCSwB, .zp-button.zp_zUY3r.zp_hLUWg.zp_n9QPr.zp_B5hnZ.zp_MCSwB.zp_IYteB",
			&buttons,
			chromedp.ByQueryAll,
			chromedp.FromNode(fromNode),
		),
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}

	var emails []string

	_ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = chromedp.Run(_ctx, chromedp.Click(buttons[0].FullXPathByID()), chromedp.PollFunction(
		`() => document.querySelectorAll(".zp_t08Bv, .zp_Pb6e2").length > 0`,
		nil,
		chromedp.WithPollingTimeout(10*time.Second),
	),
		chromedp.Evaluate(
			`(() => [].slice.call(document.querySelectorAll(".zp_t08Bv, .zp_Pb6e2")).map(e => e.innerText))()`,
			&emails,
		),
		// Hitting Escape removes focus from the email pop-up, hence closing it.
		chromedp.KeyEvent(kb.Escape),
		chromedp.WaitNotPresent(".zp_t08Bv, .zp_Pb6e2", chromedp.ByQueryAll),
	)

	return emails, err
}

func handleScrapeError(err error, field string) {
	if err != nil {
		log.Warn().Str("field", field).Err(err).Msg("failed to scrape field")
	}
}

// randomizeTTE or randomize time to email causes the current goroutine
// to sleep for a randomised duration of time before extracting the email.
func randomizeTTE() {
	low := rand.Int64N(1230-1008+1) + 1230
	high := rand.Int64N(2223-1726+1) + 1726

	dur := time.Duration(rand.Int64N(high-low+1) + low)
	time.Sleep(dur * time.Millisecond)
}

// ScrapePage scrapes all available apollo leads on the current page.
func ScrapePage(ctx context.Context, timeout time.Duration) ([]*models.Lead, error) {
	if err := PollTableData(ctx, timeout); err != nil {
		return nil, err
	}

	var rows []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes("tr.zp_cWbgJ", &rows, chromedp.ByQueryAll))
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, ErrorListEnd
	}

	leads := make([]*models.Lead, 0, len(rows))
	for i := range rows {
		row := rows[i]
		var columns []*cdp.Node
		err = chromedp.Run(
			ctx,
			chromedp.Nodes(".zp_aBhrx", &columns, chromedp.FromNode(row), chromedp.ByQueryAll),
		)
		if err != nil {
			return nil, err
		}

		var lead models.Lead

		lead.Email, err = getEmails(ctx, row)
		handleScrapeError(err, "email")
		randomizeTTE()

		name, err := getInnerText(ctx, columns[0])
		handleScrapeError(err, "name")
		lead.Name = strings.ReplaceAll(name, "\n------", "")

		lead.Title, err = getInnerText(ctx, columns[1])
		handleScrapeError(err, "title")

		lead.Company, err = getInnerText(ctx, columns[2])
		handleScrapeError(err, "company")

		lead.Phone, err = getInnerText(ctx, columns[4])
		handleScrapeError(err, "phone")

		lead.Location, err = getInnerText(ctx, columns[5])
		handleScrapeError(err, "location")

		lead.Employees, err = getInnerText(ctx, columns[6])
		handleScrapeError(err, "employees")

		lead.Industry, err = getInnerText(ctx, columns[7])
		handleScrapeError(err, "industry")

		lead.Keywords, err = getInnerText(ctx, columns[8])
		handleScrapeError(err, "keywords")

		lead.Links, err = getLinks(ctx, columns[2])
		handleScrapeError(err, "links")

		lead.Linkedin, err = getLinks(ctx, columns[0])
		handleScrapeError(err, "linkedin")

		leads = append(leads, &lead)

		log.Info().Msgf("scraped: %v/%v leads on this page", i+1, len(rows))
	}

	log.Info().Msgf("finished scraping page with: %v leads", len(rows))

	return leads, nil
}
