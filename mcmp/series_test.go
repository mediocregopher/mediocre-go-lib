package mcmp

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestSeries(t *T) {
	key := "foo"
	c := new(Component)

	assertGetElement := func(i int, expEl SeriesElement) massert.Assertion {
		el, ok := SeriesGetElement(c, key, i)
		if expEl == (SeriesElement{}) {
			return massert.Equal(false, ok)
		}
		return massert.All(
			massert.Equal(expEl, el),
			massert.Equal(true, ok),
		)
	}

	// test empty state
	massert.Require(t,
		massert.Length(SeriesElements(c, key), 0),
		massert.Length(SeriesValues(c, key), 0),
		assertGetElement(0, SeriesElement{}),
	)

	// test after a single value has been added
	AddSeriesValue(c, key, 1)
	massert.Require(t,
		massert.Equal([]SeriesElement{{Value: 1}}, SeriesElements(c, key)),
		massert.Equal([]interface{}{1}, SeriesValues(c, key)),
		assertGetElement(0, SeriesElement{Value: 1}),
		assertGetElement(1, SeriesElement{}),
	)

	// test after a child has been added
	childA := c.Child("a")
	massert.Require(t,
		massert.Equal(
			[]SeriesElement{{Value: 1}, {Child: childA}},
			SeriesElements(c, key),
		),
		massert.Equal([]interface{}{1}, SeriesValues(c, key)),
		assertGetElement(0, SeriesElement{Value: 1}),
		assertGetElement(1, SeriesElement{Child: childA}),
		assertGetElement(2, SeriesElement{}),
	)

	// test after another value has been added
	AddSeriesValue(c, key, 2)
	massert.Require(t,
		massert.Equal(
			[]SeriesElement{{Value: 1}, {Child: childA}, {Value: 2}},
			SeriesElements(c, key),
		),
		massert.Equal([]interface{}{1, 2}, SeriesValues(c, key)),
		assertGetElement(0, SeriesElement{Value: 1}),
		assertGetElement(1, SeriesElement{Child: childA}),
		assertGetElement(2, SeriesElement{Value: 2}),
		assertGetElement(3, SeriesElement{}),
	)
}
