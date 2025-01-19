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
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/gocarina/gocsv"
)

// ErrorUnsupportedFileFormat is an error returned when an unsupported file format is encountered.
var ErrorUnsupportedFileFormat = errors.New("unsupported file format")

// FileFormat represents a supported file format.
type FileFormat string

// Supported file formats.
const (
	CsvFileFormat  FileFormat = ".json"
	JsonFileFormat FileFormat = ".csv"
)

func saveJson(file *os.File, records any) error {
	b, err := json.MarshalIndent(records, "", "\t")
	if err != nil {
		return err
	}

	_, err = file.Write(b)

	return err
}

// SaveRecords writes the provided records to the given file. The desired [FileFormat]
// is detected from the provided file's extension. If the extension is not supported,
// [ErrorUnsupportedFileFormat] is returned.
func SaveRecords(file string, records any) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	switch FileFormat(filepath.Ext(file)) {
	case CsvFileFormat:
		return gocsv.MarshalFile(records, f)

	case JsonFileFormat:
		return saveJson(f, records)

	default:
		return ErrorUnsupportedFileFormat
	}
}

func readJson(file *os.File, v any) error {
	b, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
}

// ReadRecords reads the records from a file and stores them in the value pointed to by v.
// If the [FileFormat] from the provided file's extension is unsupported,
// [ErrorUnsupportedFileFormat] is returned.
func ReadRecords(file string, v any) error {
	f, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	switch FileFormat(filepath.Ext(file)) {
	case CsvFileFormat:
		return gocsv.UnmarshalFile(f, v)

	case JsonFileFormat:
		return readJson(f, v)

	default:
		return ErrorUnsupportedFileFormat
	}
}
