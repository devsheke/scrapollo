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

package models

import "time"

// TimeFormat is the time layout used by Apollo to display time.
const TimeFormat string = "Jan 02, 2006 3:04 PM"

// Time is a sugared representation of a Go [time.Time] value with
// additional methods to check if it's a zero value.
type Time struct {
	valid bool
	time  time.Time
}

// NewTime creates and returns a new instance of [*Time].
func NewTime() *Time {
	return &Time{}
}

// NewTimeValid creates and returns a new (valid) instance of [*Time].
func NewTimeValid(time time.Time) *Time {
	return &Time{valid: true, time: time}
}

// Valid returns true if the underlying time is not a zero value.
func (t *Time) Valid() bool {
	return t.valid
}

// Get returns the underlying [time.Time] value.
func (t *Time) Get() (time.Time, bool) {
	return t.time, t.valid
}

// Set sets the underlying [time.Time] value with the provided arg.
func (t *Time) Set(_t time.Time) {
	t.valid = true
	t.time = _t
}

// Increment increments the underlying [time.Time] value (if it exists)
// by the duration specified.
func (t *Time) Increment(dur time.Duration) {
	if t.valid {
		t.time = t.time.Add(dur)
	}
}

// Reset resets the underlying [time.Time] to a zero value.
func (t *Time) Reset() {
	t.valid = false
	t.time = time.Time{}
}

func (t *Time) marshal() (string, error) {
	if !t.valid {
		return "", nil
	}

	return t.time.Format(TimeFormat), nil
}

func (t *Time) unmarshal(record string) error {
	if record == "" {
		return nil
	}

	time, err := time.Parse(TimeFormat, record)
	if err != nil {
		return err
	}

	*t = Time{valid: true, time: time}

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
