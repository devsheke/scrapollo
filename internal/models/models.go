package models

import "time"

// ApolloLead represents a lead from apollo.io.
type ApolloLead struct {
	Name      string   `json:"name"      csv:"name"`
	Title     string   `json:"title"     csv:"title"`
	Company   string   `json:"company"   csv:"company"`
	Location  string   `json:"location"  csv:"location"`
	Employees string   `json:"employees" csv:"employees"`
	Phone     string   `json:"phone"     csv:"phone"`
	Industry  string   `json:"industry"  csv:"industry"`
	Keywords  string   `json:"keywords"  csv:"keywords"`
	Email     []string `json:"email"     csv:"email"`
	Links     []string `json:"links"     csv:"links"`
	Linkedin  []string `json:"linkedin"  csv:"linkedin"`
}

// ApolloAccount represents an apollo.io user in additon
// to their respective scrape status indicators.
type ApolloAccount struct {
	Email         string `csv:"email"          json:"email"`
	Password      string `csv:"password"       json:"password"`
	SaveToList    string `csv:"save-to-list"   json:"save-to-list"`
	URL           string `csv:"url"            json:"url"`
	VpnConfig     string `csv:"vpn"            json:"vpn"`
	Credits       int    `csv:"credits"        json:"credits"`
	CreditRefresh *Time  `csv:"credit-refresh" json:"credit-refresh"`
	Timeout       *Time  `csv:"timeout"        json:"timeout"`
	Target        int    `csv:"target"         json:"target"`
	Saved         int    `csv:"saved"          json:"saved"`
	done          bool   `csv:"-"`
}

// SetDone updates the ApolloAccount scrape status to reflect that
// all leads have been scraped.
func (a *ApolloAccount) SetDone() {
	a.done = true
}

// IsDone returns true if the ApolloAccount has no more new leads
// to scrape.
func (a *ApolloAccount) IsDone() bool {
	return a.Saved >= a.Target || a.done
}

// IncSaved increases the amount of saved leads.
func (a *ApolloAccount) IncSaved(amnt int) {
	a.Saved += amnt
}

// IsTimedOut returns true if the ApolloAccount has hit the daily
// limit of scraping new leads.
func (a *ApolloAccount) IsTimedOut() bool {
	if a.Timeout.ok {
		return true
	}

	cond := time.Now().Before(a.Timeout.time)
	if cond {
		return true
	}

	a.Timeout.Reset()
	return false
}

// SetTimeout sets a timeout lasting for the given duration
func (a *ApolloAccount) SetTimeout(duration time.Duration) {
	time := Time{true, time.Now().Add(duration)}
	if a.Timeout == nil {
		a.Timeout = &time
	} else {
		*a.Timeout = time
	}
}

// UseCredits updates credit usage of an ApolloAccount.
func (a *ApolloAccount) UseCredits(amnt int) {
	a.Credits -= amnt
}

// CanScrape returns true if the ApolloAccount has enough credits to scrape leads.
func (a *ApolloAccount) CanScrape() bool {
	return a.Credits > 0 || time.Now().After(a.CreditRefresh.time)
}
