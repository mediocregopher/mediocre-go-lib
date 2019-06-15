package mcmp

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestSeries(t *T) {
	key := "foo"

	// test empty state
	c := new(Component)
	massert.Require(t,
		massert.Length(GetSeriesElements(c, key), 0),
		massert.Length(GetSeriesValues(c, key), 0),
	)

	// test after a single value has been added
	AddSeriesValue(c, key, 1)
	massert.Require(t,
		massert.Equal([]SeriesElement{{Value: 1}}, GetSeriesElements(c, key)),
		massert.Equal([]interface{}{1}, GetSeriesValues(c, key)),
	)

	// test after a child has been added
	childA := c.Child("a")
	massert.Require(t,
		massert.Equal(
			[]SeriesElement{{Value: 1}, {Child: childA}},
			GetSeriesElements(c, key),
		),
		massert.Equal([]interface{}{1}, GetSeriesValues(c, key)),
	)

	// test after another value has been added
	AddSeriesValue(c, key, 2)
	massert.Require(t,
		massert.Equal(
			[]SeriesElement{{Value: 1}, {Child: childA}, {Value: 2}},
			GetSeriesElements(c, key),
		),
		massert.Equal([]interface{}{1, 2}, GetSeriesValues(c, key)),
	)
}
