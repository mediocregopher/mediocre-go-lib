package mtime

import (
	"encoding/json"
	"strconv"
	. "testing"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTS(t *T) {
	ts := NowTS()

	tsJ, err := json.Marshal(&ts)
	require.Nil(t, err)

	// tsJ should basically be an integer
	tsF, err := strconv.ParseFloat(string(tsJ), 64)
	require.Nil(t, err)
	assert.True(t, tsF > 0)

	ts2 := TSFromFloat64(tsF)
	assert.Equal(t, ts, ts2)

	var ts3 TS
	err = json.Unmarshal(tsJ, &ts3)
	require.Nil(t, err)
	assert.Equal(t, ts, ts3)
}

// Make sure that we can take in a non-float from json
func TestTSMarshalInt(t *T) {
	now := time.Now()
	tsJ := []byte(strconv.FormatInt(now.Unix(), 10))
	var ts TS
	err := json.Unmarshal(tsJ, &ts)
	require.Nil(t, err)
	assert.Equal(t, ts.Float64(), float64(now.Unix()))
}

type Foo struct {
	T TS `json:"timestamp" bson:"t"`
}

func TestTSJSON(t *T) {
	now := NowTS()
	in := Foo{now}
	b, err := json.Marshal(in)
	require.Nil(t, err)
	assert.NotEmpty(t, b)

	var out Foo
	err = json.Unmarshal(b, &out)
	require.Nil(t, err)
	assert.Equal(t, in, out)
}

func TestTSJSONNull(t *T) {
	{
		var foo Foo
		timestampNull := []byte(`{"timestamp":null}`)
		fooJSON, err := json.Marshal(foo)
		require.Nil(t, err)
		assert.Equal(t, timestampNull, fooJSON)

		require.Nil(t, json.Unmarshal(timestampNull, &foo))
		assert.True(t, foo.T.IsZero())
		assert.False(t, foo.T.IsUnixZero())
	}

	{
		var foo Foo
		foo.T = TS{Time: unixZero}
		timestampZero := []byte(`{"timestamp":0}`)
		fooJSON, err := json.Marshal(foo)
		require.Nil(t, err)
		assert.Equal(t, timestampZero, fooJSON)

		require.Nil(t, json.Unmarshal(timestampZero, &foo))
		assert.False(t, foo.T.IsZero())
		assert.True(t, foo.T.IsUnixZero())
	}
}

func TestTSZero(t *T) {
	var ts TS
	assert.True(t, ts.IsZero())
	assert.False(t, ts.IsUnixZero())
	tsf := timeToFloat(ts.Time)
	assert.Zero(t, tsf)

	ts = TSFromFloat64(0)
	assert.False(t, ts.IsZero())
	assert.True(t, ts.IsUnixZero())
	tsf = timeToFloat(ts.Time)
	assert.Zero(t, tsf)
}

func TestTSBSON(t *T) {
	// BSON only supports up to millisecond precision, but even if we keep that
	// many it kinda gets messed up due to rounding errors. So we just give it
	// one with second precision
	now := TSFromInt64(time.Now().Unix())

	in := Foo{now}
	b, err := bson.Marshal(in)
	require.Nil(t, err)
	assert.NotEmpty(t, b)

	var out Foo
	err = bson.Unmarshal(b, &out)
	require.Nil(t, err)
	assert.Equal(t, in, out)
}
