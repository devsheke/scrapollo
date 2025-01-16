// Copyright 2025 Abhisheke Acharya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package io

import (
	"encoding/json"
	"os"

	"github.com/devsheke/scrapollo/internal/models"
	"github.com/gocarina/gocsv"
)

// LeadWriter defines an interface for writing lead data. It provides
// methods to handle individual leads or a collection of leads.
type LeadWriter interface {
	// WriteLead writes a single lead to the underlying destination.
	WriteLead(*models.Lead) error

	// WriteLeads writes a collection of leads to the underlying destination.
	WriteLeads([]*models.Lead) error
}

// CsvLeadWriter is an implementation of a [LeadWriter] that writes lead data
// to a CSV file.
type CsvLeadWriter struct {
	file string
}

// NewCsvLeadWriter returns an instance of a [LeadWriter] that writes lead data
// to the given CSV file.
func NewCsvLeadWriter(file string) LeadWriter {
	return &CsvLeadWriter{file}
}

func (c *CsvLeadWriter) WriteLead(lead *models.Lead) error {
	file, err := os.OpenFile(c.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return gocsv.MarshalFile([]*models.Lead{lead}, file)
}

func (c *CsvLeadWriter) WriteLeads(leads []*models.Lead) error {
	file, err := os.OpenFile(c.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return gocsv.MarshalFile(leads, file)
}

// JsonLeadWriter is an implementation of a [LeadWriter] that writes lead data
// to a JSON file.
type JsonLeadWriter struct {
	file string
}

// NewJsonLeadWriter returns an instance of a [LeadWriter] that writes lead data
// to the given JSON file.
func NewJsonLeadWriter(file string) LeadWriter {
	return &JsonLeadWriter{file}
}

func (j *JsonLeadWriter) WriteLead(lead *models.Lead) error {
	file, err := os.OpenFile(j.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	b, err := json.Marshal(lead)
	if err != nil {
		return err
	}

	_, err = file.Write(b)

	return err
}

func (j *JsonLeadWriter) WriteLeads(leads []*models.Lead) error {
	file, err := os.OpenFile(j.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, lead := range leads {
		b, err := json.Marshal(lead)
		if err != nil {
			return err
		}

		// append a new line
		b = append(b, 10)

		if _, err = file.Write(b); err != nil {
			return err
		}
	}

	return nil
}
