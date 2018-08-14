package mcfg

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func fuzzyParse(p Param, v string) json.RawMessage {
	if p.IsBool {
		if v == "" || v == "0" || v == "false" {
			return json.RawMessage("false")
		}
		return json.RawMessage("true")

	} else if p.IsString && (v == "" || v[0] != '"') {
		return json.RawMessage(`"` + v + `"`)
	}

	return json.RawMessage(v)
}

// TODO moving Path into Param would make a lot more sense

// ParamValue describes a value for a parameter which has been parsed by a
// Source
type ParamValue struct {
	Param
	Path  []string // nil if root
	Value json.RawMessage
}

func (pv ParamValue) displayName() string {
	return strings.Join(append(pv.Path, pv.Param.Name), "-")
}

func (pv ParamValue) hash() string {
	h := md5.New()
	for _, path := range pv.Path {
		fmt.Fprintf(h, "pathEl:%q\n", path)
	}
	fmt.Fprintf(h, "name:%q\n", pv.Param.Name)
	hStr := hex.EncodeToString(h.Sum(nil))
	// we add the displayName to it to make debugging easier
	return pv.displayName() + "/" + hStr
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

// Sources combines together multiple Source instances into one. It will call
// Parse on each element individually. Later Sources take precedence over
// previous ones in the slice.
type Sources []Source

// Parse implements the method for the Source interface.
func (ss Sources) Parse(c *Cfg) ([]ParamValue, error) {
	var pvs []ParamValue
	for _, s := range ss {
		innerPVs, err := s.Parse(c)
		if err != nil {
			return nil, err
		}
		pvs = append(pvs, innerPVs...)
	}
	return pvs, nil
}

// SourceMap implements the Source interface by mapping parameter names to
// values for them. The names are comprised of the path and name of a Param
// joined by "-" characters, i.e. `strings.Join(append(p.Path, p.Name), "-")`.
// Values will be parsed in the same way that SourceEnv parses its variables.
type SourceMap map[string]string

func (m SourceMap) Parse(c *Cfg) ([]ParamValue, error) {
	pvs := make([]ParamValue, 0, len(m))
	for _, pv := range c.allParamValues() {
		if v, ok := m[pv.displayName()]; ok {
			pv.Value = fuzzyParse(pv.Param, v)
			pvs = append(pvs, pv)
		}
	}
	return pvs, nil
}
