// Package mcfg implements the creation of different types of configuration
// parameters and various methods of filling those parameters from external
// configuration sources (e.g. the command line and environment variables).
//
// Parameters are registered onto a Component, and that same Component (or one
// of its ancestors) is used later to collect and fill those parameters.
package mcfg

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
)

// TODO Sources:
// - JSON file
// - YAML file

// TODO WithCLISubCommand does not play nice with the expected use-case of
// having CLI params overwrite Env ones. If Env is specified first in the
// Sources slice then it won't know about any extra Params which might get added
// due to a sub-command, but if it's specified second then Env values will
// overwrite CLI ones.

func sortParams(params []Param) {
	sort.Slice(params, func(i, j int) bool {
		a, b := params[i], params[j]
		aPath, bPath := a.Component.Path(), b.Component.Path()
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

// CollectParams gathers all Params by recursively retrieving them from the
// given Component and its children. Returned Params are sorted according to
// their Path and Name.
func CollectParams(cmp *mcmp.Component) []Param {
	var params []Param

	var visit func(*mcmp.Component)
	visit = func(cmp *mcmp.Component) {
		for _, param := range getLocalParams(cmp) {
			params = append(params, param)
		}

		for _, childCmp := range cmp.Children() {
			visit(childCmp)
		}
	}
	visit(cmp)

	sortParams(params)
	return params
}

func paramHash(path []string, name string) string {
	h := md5.New()
	for _, pathEl := range path {
		fmt.Fprintf(h, "pathEl:%q\n", pathEl)
	}
	fmt.Fprintf(h, "name:%q\n", name)
	hStr := hex.EncodeToString(h.Sum(nil))
	// we add the displayName to it to make debugging easier
	return paramFullName(path, name) + "/" + hStr
}

// Populate uses the Source to populate the values of all Params which were
// added to the given Component, and all of its children. Populate may be called
// multiple times with the same Component, each time will only affect the values
// of the Params which were provided by the respective Source.
//
// Source may be nil to indicate that no configuration is provided. Only default
// values will be used, and if any parameters are required this will error.
//
// Populating Params can affect the Component itself, for example in the case of
// sub-commands.
func Populate(cmp *mcmp.Component, src Source) error {
	if src == nil {
		src = ParamValues(nil)
	}

	pvs, err := src.Parse(cmp)
	if err != nil {
		return err
	}

	// map Params to their hash, so we can match them to their ParamValues.
	// later. There should not be any duplicates here.
	params := CollectParams(cmp)
	pM := map[string]Param{}
	for _, p := range params {
		path := p.Component.Path()
		hash := paramHash(path, p.Name)
		if _, ok := pM[hash]; ok {
			panic("duplicate Param: " + paramFullName(path, p.Name))
		}
		pM[hash] = p
	}

	// dedupe the ParamValues based on their hashes, with the last ParamValue
	// taking precedence. Also filter out those with no corresponding Param.
	pvM := map[string]ParamValue{}
	for _, pv := range pvs {
		hash := paramHash(pv.Path, pv.Name)
		if _, ok := pM[hash]; !ok {
			continue
		}
		pvM[hash] = pv
	}

	// check for required params
	for hash, p := range pM {
		if !p.Required {
			continue
		} else if _, ok := pvM[hash]; !ok {
			ctx := mctx.Annotate(p.Component.Context(),
				"param", paramFullName(p.Component.Path(), p.Name))
			return merr.New("required parameter is not set", ctx)
		}
	}

	// do the actual populating
	for hash, pv := range pvM {
		// at this point, all ParamValues in pvM have a corresponding pM Param
		p := pM[hash]
		if err := json.Unmarshal(pv.Value, p.Into); err != nil {
			return err
		}
	}

	return nil
}
