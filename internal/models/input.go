package models

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/gocarina/gocsv"
)

// TimeFormat is the time layout which is used by Apollo
// to display credit usage.
const TimeFormat string = "Jan 02, 2006 3:04 PM MST"

// ApolloAccount represents an apollo.io user in additon
// to their respective scrape status indicators.
type ApolloAccount struct {
	Email         string `csv:"email"          json:"email"`
	Password      string `csv:"password"       json:"password"`
	List          string `csv:"list"           json:"list"`
	URL           string `csv:"url"            json:"url"`
	VpnConfig     string `csv:"vpn"            json:"vpn"`
	Credits       int    `csv:"credits"        json:"credits"`
	CreditRefresh *Time  `csv:"credit-refresh" json:"credit-refresh"`
	Timeout       *Time  `csv:"timeout"        json:"timeout"`
	Target        int    `csv:"target"         json:"target"`
	Saved         int    `csv:"saved"          json:"saved"`
	_done         bool   `csv:"-"`
}

// ReadAccountsFile parses a CSV or JSON file containing apollo.io
// account credentials and other data which corresponds to the
// ApolloAccount type.
func ReadAccountsFile(file string) ([]*ApolloAccount, error) {
	ext := filepath.Ext(file)

	var records []*ApolloAccount
	var err error

	switch ext {
	case ".json":
		records, err = readFromJSON(file)
	case ".csv":
		records, err = readFromCSV(file)
	default:
		return nil, errors.New("unknown input file format")
	}

	if err != nil {
		return nil, err
	}

	for _, record := range records {
		if record.Timeout == nil {
			record.Timeout = &Time{}
		}
		if record.CreditRefresh == nil {
			record.Timeout = &Time{}
		}
	}

	return records, nil
}

func readFromCSV(file string) ([]*ApolloAccount, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []*ApolloAccount
	err = gocsv.UnmarshalFile(f, &records)

	return records, err
}

func readFromJSON(file string) ([]*ApolloAccount, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var records []*ApolloAccount
	err = json.Unmarshal(b, &records)

	return records, err
}

// IsDone returns true if the ApolloAccount has no more new leads
// to scrape.
func (a *ApolloAccount) IsDone() bool {
	return a.Saved >= a.Target || a._done
}

// IncSaved increases the amount of saved leads.
func (a *ApolloAccount) IncSaved(amnt int) {
	a.Saved += amnt
}

// Done updates the ApolloAccount scrape state to 'Completed'.
func (a *ApolloAccount) Done() {
	a._done = true
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
	return a.Credits > 0
}

// Time is a representation of nullable library's time type.
type Time struct {
	ok   bool
	time time.Time
}

func NewTime(time time.Time) *Time {
	return &Time{true, time}
}

// Get returns the underlying time value
func (t *Time) Get() time.Time {
	return t.time
}

// Reset sets the time value to its zero equivalent.
func (t *Time) Reset() {
	t.ok = false
	t.time = time.Time{}
}

func (t *Time) marshal() (string, error) {
	if !t.ok {
		return "", nil
	}

	return t.time.Format(TimeFormat), nil
}

func (t *Time) unmarshal(record string) error {
	if record == "" {
		var _t Time
		*t = _t
		return nil
	}

	time, err := time.Parse(TimeFormat, record)
	if err != nil {
		return err
	}
	*t = Time{ok: true, time: time}

	return nil
}

func (t *Time) MarshalCSV() (string, error) {
	return t.marshal()
}

func (t *Time) UnmarshalCSV(record string) error {
	return t.unmarshal(record)
}

func (t *Time) MarshalJSON() ([]byte, error) {
	s, err := t.marshal()
	return []byte(s), err
}

func (t *Time) UnmarshalJSON(field []byte) error {
	return t.unmarshal(string(field))
}
