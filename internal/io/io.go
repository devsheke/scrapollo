package io

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/gocarina/gocsv"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

// These constants are used to decide the output format.
// Currently JSON and CSV output formats are supported.
const (
	CSVOutput int = iota
	JSONOutput
)

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

// ReadAccountsFile parses a CSV or JSON file containing apollo.io
// account credentials and other data which corresponds to the
// ApolloAccount type.
func ReadAccountsFile(file string) ([]*models.ApolloAccount, error) {
	ext := filepath.Ext(file)

	var records []*models.ApolloAccount
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
			record.Timeout = &models.Time{}
		}
		if record.CreditRefresh == nil {
			record.Timeout = &models.Time{}
		}
	}

	return records, nil
}

func readFromCSV(file string) ([]*models.ApolloAccount, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []*models.ApolloAccount
	err = gocsv.UnmarshalFile(f, &records)

	return records, err
}

func readFromJSON(file string) ([]*models.ApolloAccount, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var records []*models.ApolloAccount
	err = json.Unmarshal(b, &records)

	return records, err
}
