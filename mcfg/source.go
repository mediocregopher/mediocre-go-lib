package mcfg

import (
	"encoding/json"
)

// ParamValue describes a value for a parameter which has been parsed by a
// Source.
type ParamValue struct {
	Name  string
	Path  []string
	Value json.RawMessage
}

// Source parses ParamValues out of a particular configuration source, given a
// sorted set of possible Params to parse.
//
// Source should not return ParamValues which were not explicitly set to a value
// by the configuration source.
//
// The returned []ParamValue may contain duplicates of the same Param's value.
// in which case the later value takes precedence. It may also contain
// ParamValues which do not correspond to any of the passed in Params. These
// will be ignored in Populate.
type Source interface {
	Parse([]Param) ([]ParamValue, error)
}

// ParamValues is simply a slice of ParamValue elements, which implements Parse
// by always returning itself as-is.
type ParamValues []ParamValue

// Parse implements the method for the Source interface.
func (pvs ParamValues) Parse([]Param) ([]ParamValue, error) {
	return pvs, nil
}

// Sources combines together multiple Source instances into one. It will call
// Parse on each element individually. Values from later Sources take precedence
// over previous ones.
type Sources []Source

// Parse implements the method for the Source interface.
func (ss Sources) Parse(params []Param) ([]ParamValue, error) {
	var pvs []ParamValue
	for _, s := range ss {
		innerPVs, err := s.Parse(params)
		if err != nil {
			return nil, err
		}
		pvs = append(pvs, innerPVs...)
	}
	return pvs, nil
}
