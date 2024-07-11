package models

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/gocarina/gocsv"
)

// These constants are used to decide the output format.
// Currently JSON and CSV output formats are supported.
const (
	CSVOutput int = iota
	JSONOutput
)

// Lead represents an apollo.io lead.
type Lead struct {
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

func ExtensionFromOutputType(o int) (string, error) {
	switch o {
	case CSVOutput:
		return ".csv", nil
	case JSONOutput:
		return ".json", nil
	default:
		return "", errors.ErrUnsupported
	}
}

// SaveRecordsToFile saves a slice of records to a file. JSON and CSV formats are supported.
func SaveRecordsToFile(records any, file string, outType int) error {
	if filepath.Ext(file) == "" {
		ext, err := ExtensionFromOutputType(outType)
		if err != nil {
			return err
		}
		file = file + ext
	}
	switch outType {
	case CSVOutput:
		return saveToCSV(records, file)
	case JSONOutput:
		return saveToJSON(records, file)
	default:
		return errors.New("unknown output filetype")
	}
}

func saveToCSV(records any, file string) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return gocsv.MarshalFile(records, f)
}

func saveToJSON(records any, file string) error {
	b, err := json.Marshal(records)
	if err != nil {
		return err
	}

	return os.WriteFile(file, b, 0644)
}
