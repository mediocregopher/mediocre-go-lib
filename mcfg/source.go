package mcfg

import "encoding/json"

// ParamValue describes a value for a parameter which has been parsed by a
// Source
type ParamValue struct {
	Param
	Path  []string // nil if root
	Value json.RawMessage
}

func (cfg *Cfg) allParamValues() []ParamValue {
	pvs := make([]ParamValue, 0, len(cfg.Params))
	for _, param := range cfg.Params {
		pvs = append(pvs, ParamValue{
			Param: param,
			Path:  cfg.Path,
		})
	}

	for _, child := range cfg.Children {
		pvs = append(pvs, child.allParamValues()...)
	}
	return pvs
}

// Source parses ParamValues out of a particular configuration source. The
// returned []ParamValue may contain duplicates of the same Param's value.
type Source interface {
	Parse(*Cfg) ([]ParamValue, error)
}
