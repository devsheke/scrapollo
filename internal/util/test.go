package util

import (
	"os"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

// GetAccountFromEnv constructs an ApolloAccount from the environment.
// This function searches the environment for 'EMAIL' and 'PASSWORD'.
// variables and fails the test if neither of them are present.
func GetAccountFromEnv(t *testing.T) *models.ApolloAccount {
	email, ok := os.LookupEnv("EMAIL")
	if !ok {
		t.Fatalf("'EMAIL' not found in env")
	}

	password, ok := os.LookupEnv("PASSWORD")
	if !ok {
		t.Fatalf("'PASSWORD' not found in env")
	}

	return &models.ApolloAccount{Email: email, Password: password}
}

// SetupBrowserFromEnv launches a browser instance using the path to a browser
// specified by the 'BROWSER' environment variable. The browser's 'headless' mode
// is controlled by the 'HEADLESS' environment variable; either 'true' or 'false'.
func SetupBrowserFromEnv(t *testing.T) *rod.Browser {
	headless := true
	switch h := os.Getenv("HEADLESS"); h {
	case "false":
		headless = false
	case "true":
		break
	default:
		t.Logf("unknown option for 'HEADLESS': %q; defaulting to 'true'.", h)
	}

	l := launcher.New().Bin(os.Getenv("BROWSER"))
	var u string
	if headless {
		u = l.Set("headless").MustLaunch()
	} else {
		u = l.Delete("--headless").MustLaunch()
	}

	return rod.New().ControlURL(u).MustConnect()
}
