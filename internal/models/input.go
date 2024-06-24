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

type ApolloAccount struct {
	Email         string `csv:"email"          json:"email"`
	Password      string `csv:"password"       json:"password"`
	VpnConfig     string `csv:"vpn"            json:"vpn"`
	Credits       int    `csv:"credits"        json:"credits"`
	CreditRefresh *Time  `csv:"credit-refresh" json:"credit-refresh"`
	Timeout       *Time  `csv:"timeout"        json:"timeout"`
	Target        int    `csv:"target"         json:"target"`
	Saved         int    `csv:"saved"          json:"saved"`
}

func ReadInput(file string) ([]*ApolloAccount, error) {
	ext := filepath.Ext(file)

	var records []*ApolloAccount
	var err error

	switch ext {
	case "json":
		records, err = readFromJSON(file)
	case "csv":
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

func (a *ApolloAccount) Done() bool {
	return a.Saved >= a.Target
}

type Time struct {
	ok   bool
	time time.Time
}

func (t *Time) Reset() {
	t.ok = false
	t.time = time.Time{}
}

func (t *Time) IsTimedOut() bool {
	if !t.ok {
		return false
	}

	cond := time.Now().Before(t.time)
	if cond {
		return cond
	}
	t.time = time.Time{}
	return false
}

func (t *Time) SetTime(_t time.Time) {
	t.ok = true
	t.time = _t
}

func (t *Time) GetTime() (time.Time, bool) {
	if t.ok {
		return t.time, t.ok
	}
	return time.Time{}, t.ok
}

func (t *Time) marshal() (string, error) {
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
