package mcfg

import (
	"encoding/json"
	"fmt"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

// The tests for the different Sources use mchk as their primary method of
// checking. They end up sharing a lot of the same functionality, so in here is
// all the code they share

type srcCommonState struct {
	// availCmps get updated in place as the run goes on, it's easier to keep
	// track of them this way than by traversing the hierarchy.
	availCmps []*mcmp.Component

	expPVs []ParamValue
	// each specific test should wrap this to add the Source itself
}

func newSrcCommonState() srcCommonState {
	var scs srcCommonState
	{
		root := new(mcmp.Component)
		a := root.Child("a")
		b := root.Child("b")
		c := root.Child("c")
		ab := a.Child("b")
		bc := b.Child("c")
		abc := ab.Child("c")
		scs.availCmps = []*mcmp.Component{root, a, b, c, ab, bc, abc}
	}
	return scs
}

type srcCommonParams struct {
	name        string
	cmp         *mcmp.Component
	isBool      bool
	nonBoolType string // "int", "str", "duration", "json"
	unset       bool
	nonBoolVal  string
}

func (scs srcCommonState) next() srcCommonParams {
	var p srcCommonParams
	if i := mrand.Intn(8); i == 0 {
		p.name = mrand.Hex(1) + "-" + mrand.Hex(8)
	} else {
		p.name = mrand.Hex(8)
	}

	availCmpI := mrand.Intn(len(scs.availCmps))
	p.cmp = scs.availCmps[availCmpI]

	p.isBool = mrand.Intn(8) == 0
	if !p.isBool {
		p.nonBoolType = mrand.Element([]string{
			"int",
			"str",
			"duration",
			"json",
		}, nil).(string)
	}
	p.unset = mrand.Intn(10) == 0

	if p.isBool || p.unset {
		return p
	}

	switch p.nonBoolType {
	case "int":
		p.nonBoolVal = fmt.Sprint(mrand.Int())
	case "str":
		p.nonBoolVal = mrand.Hex(16)
	case "duration":
		dur := time.Duration(mrand.Intn(86400)) * time.Second
		p.nonBoolVal = dur.String()
	case "json":
		b, _ := json.Marshal(map[string]int{
			mrand.Hex(4): mrand.Int(),
			mrand.Hex(4): mrand.Int(),
			mrand.Hex(4): mrand.Int(),
		})
		p.nonBoolVal = string(b)
	}
	return p
}

// adds the new param to the cmp, and if the param is expected to be set in
// the Source adds it to the expected ParamValues as well
func (scs srcCommonState) applyCmpAndPV(p srcCommonParams) srcCommonState {
	param := Param{
		Name:     p.name,
		IsString: p.nonBoolType == "str" || p.nonBoolType == "duration",
		IsBool:   p.isBool,
		// the Sources don't actually care about the other fields of Param,
		// those are only used by Populate once it has all ParamValues together
	}
	AddParam(p.cmp, param)
	param, _ = getParam(p.cmp, param.Name) // get it back out to get any added fields

	if !p.unset {
		pv := ParamValue{Name: param.Name, Path: p.cmp.Path()}
		if p.isBool {
			pv.Value = json.RawMessage("true")
		} else {
			switch p.nonBoolType {
			case "str", "duration":
				pv.Value = json.RawMessage(fmt.Sprintf("%q", p.nonBoolVal))
			case "int", "json":
				pv.Value = json.RawMessage(p.nonBoolVal)
			default:
				panic("shouldn't get here")
			}
		}
		scs.expPVs = append(scs.expPVs, pv)
	}

	return scs
}

// given a Source asserts that it's Parse method returns the expected
// ParamValues
func (scs srcCommonState) assert(s Source) error {
	gotPVs, err := s.Parse(scs.availCmps[0]) // Parse(root)
	if err != nil {
		return err
	}
	return massert.All(
		massert.Length(gotPVs, len(scs.expPVs)),
		massert.Subset(scs.expPVs, gotPVs),
	).Assert()
}

func TestSources(t *T) {
	cmp := new(mcmp.Component)
	a := Int(cmp, "a", ParamRequired())
	b := Int(cmp, "b", ParamRequired())
	c := Int(cmp, "c", ParamRequired())

	err := Populate(cmp, Sources{
		&SourceCLI{Args: []string{"--a=1", "--b=666"}},
		&SourceEnv{Env: []string{"B=2", "C=3"}},
	})
	massert.Require(t,
		massert.Nil(err),
		massert.Equal(1, *a),
		massert.Equal(2, *b),
		massert.Equal(3, *c),
	)
}

func TestSourceParamValues(t *T) {
	cmp := new(mcmp.Component)
	a := Int(cmp, "a", ParamRequired())
	cmpFoo := cmp.Child("foo")
	b := String(cmpFoo, "b", ParamRequired())
	c := Bool(cmpFoo, "c")

	err := Populate(cmp, ParamValues{
		{Name: "a", Value: json.RawMessage(`4`)},
		{Path: []string{"foo"}, Name: "b", Value: json.RawMessage(`"bbb"`)},
		{Path: []string{"foo"}, Name: "c", Value: json.RawMessage("true")},
	})
	massert.Require(t,
		massert.Nil(err),
		massert.Equal(4, *a),
		massert.Equal("bbb", *b),
		massert.Equal(true, *c),
	)
}
