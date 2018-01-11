package mtime

// Code based off the timeutil package in github.com/levenlabs/golib
// Changes performed:
// - Renamed Timestamp to TS for brevity
// - Added NewTS function
// - Moved Float64 method
// - Moved initialization methods to top
// - Made MarshalJSON use String method
// - TSNow -> NowTS, make it use NewTS

import (
	"bytes"
	"strconv"
	"time"
)

var unixZero = time.Unix(0, 0)

func timeToFloat(t time.Time) float64 {
	// If time.Time is the empty value, UnixNano will return the farthest back
	// timestamp a float can represent, which is some large negative value. We
	// compromise and call it zero
	if t.IsZero() {
		return 0
	}
	return float64(t.UnixNano()) / 1e9
}

// TS is a wrapper around time.Time which adds methods to marshal and
// unmarshal the value as a unix timestamp instead of a formatted string
type TS struct {
	time.Time
}

// NewTS returns a new TS instance wrapping the given time.Time, which will
// possibly be truncated a certain amount to account for floating point
// precision.
func NewTS(t time.Time) TS {
	return TSFromFloat64(timeToFloat(t))
}

// NowTS is a wrapper around time.Now which returns a TS.
func NowTS() TS {
	return NewTS(time.Now())
}

// TSFromInt64 returns a TS equal to the given int64, assuming it too is a unix
// timestamp
func TSFromInt64(ts int64) TS {
	return TS{time.Unix(ts, 0)}
}

// TSFromFloat64 returns a TS equal to the given float64, assuming it too is a
// unix timestamp. The float64 is interpreted as number of seconds, with
// everything after the decimal indicating milliseconds, microseconds, and
// nanoseconds
func TSFromFloat64(ts float64) TS {
	secs := int64(ts)
	nsecs := int64((ts - float64(secs)) * 1e9)
	return TS{time.Unix(secs, nsecs)}
}

// TSFromString attempts to parse the string as a float64, and then passes that
// into TSFromFloat64, returning the result
func TSFromString(ts string) (TS, error) {
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return TS{}, err
	}
	return TSFromFloat64(f), nil
}

// String returns the string representation of the TS, in the form of a floating
// point form of the time as a unix timestamp
func (t TS) String() string {
	ts := timeToFloat(t.Time)
	return strconv.FormatFloat(ts, 'f', -1, 64)
}

// Float64 returns the float representation of the timestamp in seconds.
func (t TS) Float64() float64 {
	return timeToFloat(t.Time)
}

var jsonNull = []byte("null")

// MarshalJSON returns the JSON representation of the TS as an integer.  It
// never returns an error
func (t TS) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return jsonNull, nil
	}

	return []byte(t.String()), nil
}

// UnmarshalJSON takes a JSON integer and converts it into a TS, or
// returns an error if this can't be done
func (t *TS) UnmarshalJSON(b []byte) error {
	// since 0 is a valid timestamp we can't use that to mean "unset", so we
	// take null to mean unset instead
	if bytes.Equal(b, jsonNull) {
		t.Time = time.Time{}
		return nil
	}

	var err error
	*t, err = TSFromString(string(b))
	return err
}

// IsUnixZero returns true if the timestamp is equal to the unix zero timestamp,
// representing 1/1/1970. This is different than checking if the timestamp is
// the empty value (which should be done with IsZero)
func (t TS) IsUnixZero() bool {
	return t.Equal(unixZero)
}
