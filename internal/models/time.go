package models

import "time"

// TimeFormat is the time layout which is used by Apollo
// to display credit usage.
const TimeFormat string = "Jan 02, 2006 3:04 PM"

// Time is a representation of nullable library's time type.
type Time struct {
	ok   bool
	time time.Time
}

// Create a new instance of *Time.
func NewTime(time time.Time) *Time {
	return &Time{true, time}
}

// IsSome returns true if this instance of *Time has been initialized.
func (t *Time) IsSome() bool {
	return t.ok
}

// Set is used to initialize or update the *Time value.
func (t *Time) Set(time time.Time) {
	t.ok = true
	t.time = time
}

// Get returns the underlying time value.
func (t *Time) Get() time.Time {
	return t.time
}

// Reset sets the time value to its zero equivalent.
func (t *Time) Reset() {
	t.ok = false
	t.time = time.Time{}
}

func (t *Time) marshal() (string, error) {
	if !t.ok {
		return "", nil
	}

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

// MarshalCSV converts the *Time value to its CSV representation.
func (t *Time) MarshalCSV() (string, error) {
	return t.marshal()
}

// UnmarshalCSV converts a CSV string to a *Time value.
func (t *Time) UnmarshalCSV(record string) error {
	return t.unmarshal(record)
}

// MarshalCSV converts the *Time value to its JSON representation.
func (t *Time) MarshalJSON() ([]byte, error) {
	s, err := t.marshal()
	return []byte(s), err
}

// UnmarshalCSV converts JSON data to a *Time value.
func (t *Time) UnmarshalJSON(field []byte) error {
	return t.unmarshal(string(field))
}
