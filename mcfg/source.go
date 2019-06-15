package mcfg

import (
	"encoding/json"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
)

// ParamValue describes a value for a parameter which has been parsed by a
// Source.
type ParamValue struct {
	Name  string
	Path  []string
	Value json.RawMessage
}

// Source parses ParamValues out of a particular configuration source, given the
// Component which the Params were added to (via WithInt, WithString, etc...).
// CollectParams can be used to retrieve these Params.
//
// It's possible for Parsing to affect the Component itself, for example in the
// case of sub-commands.
//
// Source should not return ParamValues which were not explicitly set to a value
// by the configuration source.
//
// The returned []ParamValue may contain duplicates of the same Param's value.
// in which case the latter value takes precedence. It may also contain
// ParamValues which do not correspond to any of the passed in Params. These
// will be ignored in Populate.
type Source interface {
	Parse(*mcmp.Component) ([]ParamValue, error)
}

// ParamValues is simply a slice of ParamValue elements, which implements Parse
// by always returning itself as-is.
type ParamValues []ParamValue

var _ Source = ParamValues{}

// Parse implements the method for the Source interface.
func (pvs ParamValues) Parse(*mcmp.Component) ([]ParamValue, error) {
	return pvs, nil
}

// Sources combines together multiple Source instances into one. It will call
// Parse on each element individually. Values from later Sources take precedence
// over previous ones.
type Sources []Source

var _ Source = Sources{}

// Parse implements the method for the Source interface.
func (ss Sources) Parse(cmp *mcmp.Component) ([]ParamValue, error) {
	var pvs []ParamValue
	for _, s := range ss {
		var innerPVs []ParamValue
		var err error
		if innerPVs, err = s.Parse(cmp); err != nil {
			return nil, err
		}
		pvs = append(pvs, innerPVs...)
	}
	return pvs, nil
}
