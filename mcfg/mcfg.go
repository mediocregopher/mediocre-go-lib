// Package mcfg provides a simple foundation for complex service/binary
// configuration, initialization, and destruction
package mcfg

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
)

// TODO Sources:
// - JSON file
// - YAML file

type ctxCfg struct {
	path   []string
	params map[string]Param
}

type ctxKey int

func get(ctx mctx.Context) *ctxCfg {
	return mctx.GetSetMutableValue(ctx, true, ctxKey(0),
		func(interface{}) interface{} {
			return &ctxCfg{
				path:   mctx.Path(ctx),
				params: map[string]Param{},
			}
		},
	).(*ctxCfg)
}

func sortParams(params []Param) {
	sort.Slice(params, func(i, j int) bool {
		a, b := params[i], params[j]
		aPath, bPath := a.Path, b.Path
		for {
			switch {
			case len(aPath) == 0 && len(bPath) == 0:
				return a.Name < b.Name
			case len(aPath) == 0 && len(bPath) > 0:
				return false
			case len(aPath) > 0 && len(bPath) == 0:
				return true
			case aPath[0] != bPath[0]:
				return aPath[0] < bPath[0]
			default:
				aPath, bPath = aPath[1:], bPath[1:]
			}
		}
	})
}

// returns all Params gathered by recursively retrieving them from this Context
// and its children. Returned Params are sorted according to their Path and
// Name.
func collectParams(ctx mctx.Context) []Param {
	var params []Param

	var visit func(mctx.Context)
	visit = func(ctx mctx.Context) {
		for _, param := range get(ctx).params {
			params = append(params, param)
		}

		for _, childCtx := range mctx.Children(ctx) {
			visit(childCtx)
		}
	}
	visit(ctx)

	sortParams(params)
	return params
}

func populate(params []Param, src Source) error {
	pvs, err := src.Parse(params)
	if err != nil {
		return err
	}

	// dedupe the ParamValues based on their hashes, with the last ParamValue
	// taking precedence
	pvM := map[string]ParamValue{}
	for _, pv := range pvs {
		pvM[pv.hash()] = pv
	}

	// check for required params
	for _, param := range params {
		if !param.Required {
			continue
		} else if _, ok := pvM[param.hash()]; !ok {
			return fmt.Errorf("param %q is required", param.fullName())
		}
	}

	for _, pv := range pvM {
		if err := json.Unmarshal(pv.Value, pv.Into); err != nil {
			return err
		}
	}

	return nil
}

// Populate uses the Source to populate the values of all Params which were
// added to the given mctx.Context, and all of its children.
func Populate(ctx mctx.Context, src Source) error {
	return populate(collectParams(ctx), src)
}
