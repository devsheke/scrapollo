package io

import (
	"math/rand/v2"
	"path/filepath"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/shadowbizz/apollo-crawler/internal/models"
)

func TestJSONWriter(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "lead_writer_test.json")
	w := NewJSONLeadWriter(filename)

	leads := make([]*models.ApolloLead, 20)
	faker := gofakeit.New(rand.Uint64())
	for i := range 20 {
		leads[i] = models.GenerateakeLead(faker)
	}

	size := len(leads)
	for {
		if size == 0 {
			break
		}

		switch rand.IntN(2) {
		case 0:
			lead := leads[size-1]
			if err := w.WriteLead(lead); err != nil {
				t.Fatal(err)
			}
			size--
		default:
			_size := rand.IntN(size)
			if err := w.WriteLeads(leads[size-1-_size:]); err != nil {
				t.Fatal(err)
			}
			size -= _size
		}
	}
}

func TestCSVWriter(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "lead_writer_test.json")
	w := NewCSVLeadWriter(filename)

	leads := make([]*models.ApolloLead, 20)
	faker := gofakeit.New(rand.Uint64())
	for i := range 20 {
		leads[i] = models.GenerateakeLead(faker)
	}

	size := len(leads)
	for {
		if size == 0 {
			break
		}

		switch rand.IntN(2) {
		case 0:
			lead := leads[size-1]
			if err := w.WriteLead(lead); err != nil {
				t.Fatal(err)
			}
			size--
		default:
			_size := rand.IntN(size)
			if err := w.WriteLeads(leads[size-1-_size:]); err != nil {
				t.Fatal(err)
			}
			size -= _size
		}
	}
}
