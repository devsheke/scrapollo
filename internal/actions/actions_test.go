package actions

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/shadowbizz/apollo-crawler/internal/models"
	"github.com/shadowbizz/apollo-crawler/internal/util"
)

var timeouts = []time.Duration{60 * time.Second, 120 * time.Second}

func TestLogin(t *testing.T) {
	scraper := util.GetAccountFromEnv(t)
	browser := util.SetupBrowserFromEnv(t)

	if _, err := LoginToApollo(browser, scraper, "/tmp", timeouts[0]); err != nil {
		t.Error(err)
	}

	if _, ok := scraper.GetLoginCookies(); !ok {
		t.Errorf("did not find any saved session cookies after logging in")
	}
}

func TestSave(t *testing.T) {
	scraper := util.GetAccountFromEnv(t)
	browser := util.SetupBrowserFromEnv(t)

	page, err := LoginToApollo(browser, scraper, "/tmp", timeouts[0])
	if err != nil {
		t.Error(err)
	}

	page.MustNavigate(os.Getenv("URL")).MustWaitDOMStable()

	start := time.Now()
	numLeads, err := SaveLeadsToList(
		page,
		fmt.Sprintf("test-%s", time.Now().Format(time.RFC3339Nano)),
		timeouts[0],
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("took %.2fs to fetch %d leads", time.Since(start).Seconds(), numLeads)
}

func TestScrape(t *testing.T) {
	scraper := util.GetAccountFromEnv(t)
	browser := util.SetupBrowserFromEnv(t)

	page, err := LoginToApollo(browser, scraper, "/tmp", timeouts[0])
	if err != nil {
		t.Fatal(err)
	}

	page.MustNavigate(os.Getenv("URL")).MustWaitDOMStable()

	start := time.Now()
	leads, err := ScrapeLeads(page, NetNewTab, timeouts[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("took %.2fs to fetch %d leads", time.Since(start).Seconds(), len(leads))

	for _, lead := range leads {
		if len(lead.Email) == 0 {
			t.Fatalf("lead is missing emails:\n%+v", lead)
		}

		if lead.Name == "" {
			t.Fatalf("lead is missing name:\n%+v", lead)
		}
	}
}

func TestGetPageInfo(t *testing.T) {
	scraper := util.GetAccountFromEnv(t)
	browser := util.SetupBrowserFromEnv(t)

	page, err := LoginToApollo(browser, scraper, "/tmp", timeouts[0])
	if err != nil {
		t.Fatal(err)
	}

	page.MustNavigate(os.Getenv("URL")).MustWaitDOMStable()

	p, err := GetPageInfo(page)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("got page info: %+v\n", p)
}

func TestNextPage(t *testing.T) {
	scraper := util.GetAccountFromEnv(t)
	browser := util.SetupBrowserFromEnv(t)

	page, err := LoginToApollo(browser, scraper, "/tmp", timeouts[0])
	if err != nil {
		t.Fatal(err)
	}

	page.MustNavigate(os.Getenv("URL")).MustWaitDOMStable()

	p, err := GetPageInfo(page)
	if err != nil {
		t.Fatal(err)
	}

	if p.IsLastPage {
		t.Fatalf("on the last page. cannot go to next page!")
	}

	err = GoToNextPage(page)
	if err != nil {
		t.Fatal(err)
	}

	n, err := GetPageInfo(page)
	if err != nil {
		t.Fatal(err)
	}

	if p.Start == n.Start || p.End == n.End {
		t.Fatalf("previous PageInfo == curret PageInfo")
	}
}

func TestFetchCredits(t *testing.T) {
	scraper := util.GetAccountFromEnv(t)
	browser := util.SetupBrowserFromEnv(t)

	page, err := LoginToApollo(browser, scraper, "/tmp", timeouts[0])
	if err != nil {
		t.Fatal(err)
	}

	scraper.Credits, scraper.CreditRefresh, err = FetchCreditUsage(page, scraper, timeouts[0])
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf(
		"credits: %d, renewal time: %s\n",
		scraper.Credits,
		scraper.CreditRefresh.Get().Format(models.TimeFormat),
	)
}

func TestGoToList(t *testing.T) {
	scraper := util.GetAccountFromEnv(t)
	browser := util.SetupBrowserFromEnv(t)

	page, err := LoginToApollo(browser, scraper, "/tmp", timeouts[0])
	if err != nil {
		page.MustScreenshot("test-gotolist.png")
		t.Fatal(err)
	}

	if err := page.Navigate(os.Getenv("URL")); err != nil {
		t.Fatal(err)
	}

	list, ok := os.LookupEnv("LIST")
	if !ok {
		t.Fatalf("could not find 'LIST' in env")
	}

	if err := GoToList(page, list, timeouts[0]); err != nil {
		page.MustScreenshot("test-gotolist.png")
		t.Fatal(err)
	}
}
