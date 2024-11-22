package models

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/gocarina/gocsv"
)

func TestInitEmptyCSVFile(t *testing.T) {
	file, err := os.OpenFile("test.csv", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if err := gocsv.MarshalFile([]*ApolloAccount{}, file); err != nil {
		t.Fatal(err)
	}

	if b, err := os.ReadFile(file.Name()); err != nil {
		t.Fatal(err)
	} else {
		fmt.Println(string(b))
	}
}

func TestInitEmptyJSONFile(t *testing.T) {
	file, err := os.OpenFile("test.json", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	b, err := json.MarshalIndent([]*ApolloAccount{&ApolloAccount{}}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := file.Write(b); err != nil {
		t.Fatal(err)
	}
}
