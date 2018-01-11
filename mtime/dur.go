package mtime

import (
	"encoding/json"
	"time"
)

// Duration wraps time.Duration to implement marshaling and unmarshaling methods
type Duration struct {
	time.Duration
}

// MarshalText implements the text.Marshaler interface
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// UnmarshalText implements the text.Unmarshaler interface
func (d *Duration) UnmarshalText(b []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(b))
	return err
}

// MarshalJSON implements the json.Marshaler interface, marshaling the Duration
// as a json string via Duration's String method
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface, unmarshaling the
// Duration as a JSON string and using the time.ParseDuration function on that
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	d.Duration, err = time.ParseDuration(s)
	return err
}
