package io

import (
	"encoding/json"
	"os"

	"github.com/gocarina/gocsv"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

const (
	appendFlag       = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	newlineByte byte = 10
	writePerm        = 0644
)

var (
	CSVLeadWriterKind  = "CSV Writer"
	JSONLeadWriterKind = "JSON Writer"
)

// LeadWriter is a type that writes models.ApolloLead(s) to
// a specified output.
type LeadWriter interface {
	Kind() string
	WriteLead(*models.ApolloLead) error
	WriteLeads([]*models.ApolloLead) error
}

type CSVLeadWriter struct {
	file string
}

func NewCSVLeadWriter(file string) LeadWriter {
	return &CSVLeadWriter{file}
}

func writeCSVToFile(filename string, leads ...*models.ApolloLead) error {
	f, err := os.OpenFile(filename, appendFlag, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	if stat.Size() == 0 {
		err = gocsv.MarshalFile(leads, f)
	} else {
		err = gocsv.MarshalWithoutHeaders(leads, f)
	}

	return err
}

func (c *CSVLeadWriter) WriteLead(lead *models.ApolloLead) error {
	return writeCSVToFile(c.file, lead)
}

func (c *CSVLeadWriter) WriteLeads(leads []*models.ApolloLead) error {
	return writeCSVToFile(c.file, leads...)
}

func (c *CSVLeadWriter) Kind() string {
	return CSVLeadWriterKind
}

type JSONLeadWriter struct {
	file string
}

func NewJSONLeadWriter(file string) LeadWriter {
	return &JSONLeadWriter{file}
}

func writeJsonToFile(f *os.File, lead *models.ApolloLead) error {
	b, err := json.Marshal(lead)
	if err != nil {
		return err
	}
	b = append(b, newlineByte)

	_, err = f.Write(b)
	return err
}

func (j *JSONLeadWriter) WriteLead(lead *models.ApolloLead) error {
	f, err := os.OpenFile(j.file, appendFlag, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return writeJsonToFile(f, lead)
}

func (j *JSONLeadWriter) WriteLeads(leads []*models.ApolloLead) error {
	f, err := os.OpenFile(j.file, appendFlag, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, lead := range leads {
		if err := writeJsonToFile(f, lead); err != nil {
			return err
		}
	}

	return nil
}

func (j *JSONLeadWriter) Kind() string {
	return JSONLeadWriterKind
}
