package mcfg

import (
	"encoding/json"
)

// ParamValue describes a value for a parameter which has been parsed by a
// Source.
type ParamValue struct {
	Param
	Value json.RawMessage
}

// Source parses ParamValues out of a particular configuration source, given a
// sorted set of possible Params to parse. The returned []ParamValue may contain
// duplicates of the same Param's value. in which case the later value takes
// precedence.
type Source interface {
	Parse([]Param) ([]ParamValue, error)
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

// SourceMap implements the Source interface by mapping parameter names to
// values for them. The names are comprised of the path and name of a Param
// joined by "-" characters, i.e. `strings.Join(append(param.Path, param.Name),
// "-")`.  Values will be parsed in the same way that SourceEnv parses its
// variables.
type SourceMap map[string]string

// Parse implements the method for the Source interface.
func (m SourceMap) Parse(params []Param) ([]ParamValue, error) {
	pvs := make([]ParamValue, 0, len(m))
	for _, p := range params {
		if v, ok := m[p.fullName()]; ok {
			pvs = append(pvs, ParamValue{
				Param: p,
				Value: p.fuzzyParse(v),
			})
		}
	}
	return pvs, nil
}
