package models

import (
	"encoding/json"
	"os"

	"github.com/gocarina/gocsv"
)

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
	Linkedin  string   `json:"linkedin"  csv:"linkedin"`
}

func SaveLeadsToFile(leads []*Lead, file string, json bool) error {
	if json {
		return saveToJSON(leads, file)
	}

	return saveToCSV(leads, file)
}

func saveToCSV(leads []*Lead, file string) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return gocsv.MarshalFile(leads, f)
}

func saveToJSON(leads []*Lead, file string) error {
	b, err := json.Marshal(leads)
	if err != nil {
		return err
	}

	return os.WriteFile(file, b, 0644)
}
