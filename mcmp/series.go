package mcmp

const (
	seriesEls int = iota
	seriesNumValueEls
)

type seriesKey struct {
	userKey interface{}
	mod     int
}

// SeriesElement is used to describe a single element in a series, as
// implemented by AddSeriesValue. A SeriesElement can either be a Child which
// was spawned from the Component, or a Value which was added via
// AddSeriesValue.
type SeriesElement struct {
	Child *Component
	Value interface{}
}

func seriesKeys(key interface{}) (seriesKey, seriesKey) {
	return seriesKey{userKey: key, mod: seriesEls},
		seriesKey{userKey: key, mod: seriesNumValueEls}
}

func getSeriesElements(c *Component, key interface{}) ([]SeriesElement, int) {
	elsKey, numValueElsKey := seriesKeys(key)
	lastEls, _ := c.Value(elsKey).([]SeriesElement)
	lastNumValueEls, _ := c.Value(numValueElsKey).(int)

	children := c.Children()
	lastNumChildrenEls := len(lastEls) - lastNumValueEls

	els := lastEls
	for _, child := range children[lastNumChildrenEls:] {
		els = append(els, SeriesElement{Child: child})
	}
	return els, lastNumValueEls
}

// AddSeriesValue is a helper which adds a value to a series which is being
// stored under the given key on the given Component. The series of values added
// under any key can be retrieved with GetSeriesValues.
//
// Additionally, AddSeriesValue keeps track of the order of calls to itself and
// children spawned from the Component. By using GetSeriesElements you can
// retrieve the sequence of values and children in the order they were added to
// the Component.
func AddSeriesValue(c *Component, key, value interface{}) {
	lastEls, lastNumValueEls := getSeriesElements(c, key)
	els := append(lastEls, SeriesElement{Value: value})

	elsKey, numValueElsKey := seriesKeys(key)
	c.SetValue(elsKey, els)
	c.SetValue(numValueElsKey, lastNumValueEls+1)
}

// SeriesElements returns the sequence of values that have been added to the
// Component under the given key via AddSeriesValue, interlaced with children
// which have been spawned from the Component, in the same respective order the
// events originally happened.
func SeriesElements(c *Component, key interface{}) []SeriesElement {
	els, _ := getSeriesElements(c, key)
	return els
}

// SeriesGetElement returns the ith element in the series at the given key.
func SeriesGetElement(c *Component, key interface{}, i int) (SeriesElement, bool) {
	els, _ := getSeriesElements(c, key)
	if i >= len(els) {
		return SeriesElement{}, false
	}
	return els[i], true
}

// SeriesValues returns the sequence of values that have been added to the
// Component under the given key via AddSeriesValue, in the same order the
// values were added.
func SeriesValues(c *Component, key interface{}) []interface{} {
	elsKey, numValueElsKey := seriesKeys(key)
	els, _ := c.Value(elsKey).([]SeriesElement)
	numValueEls, _ := c.Value(numValueElsKey).(int)

	values := make([]interface{}, 0, numValueEls)
	for _, el := range els {
		if el.Child != nil {
			continue
		}
		values = append(values, el.Value)
	}
	return values
}
