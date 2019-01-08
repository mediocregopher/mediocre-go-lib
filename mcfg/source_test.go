package mcfg

import (
	"encoding/json"
	"fmt"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

// The tests for the different Sources use mchk as their primary method of
// checking. They end up sharing a lot of the same functionality, so in here is
// all the code they share

type srcCommonState struct {
	ctx       mctx.Context
	availCtxs []mctx.Context
	expPVs    []ParamValue

	// each specific test should wrap this to add the Source itself
}

func newSrcCommonState() srcCommonState {
	var scs srcCommonState
	scs.ctx = mctx.New()
	{
		a := mctx.ChildOf(scs.ctx, "a")
		b := mctx.ChildOf(scs.ctx, "b")
		c := mctx.ChildOf(scs.ctx, "c")
		ab := mctx.ChildOf(a, "b")
		bc := mctx.ChildOf(b, "c")
		abc := mctx.ChildOf(ab, "c")
		scs.availCtxs = []mctx.Context{scs.ctx, a, b, c, ab, bc, abc}
	}
	return scs
}

type srcCommonParams struct {
	name        string
	availCtxI   int // not technically needed, but makes finding the ctx easier
	path        []string
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

	p.availCtxI = mrand.Intn(len(scs.availCtxs))
	thisCtx := scs.availCtxs[p.availCtxI]
	p.path = mctx.Path(thisCtx)

	p.isBool = mrand.Intn(2) == 0
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

// adds the new param to the ctx, and if the param is expected to be set in
// the Source adds it to the expected ParamValues as well
func (scs srcCommonState) applyCtxAndPV(p srcCommonParams) srcCommonState {
	thisCtx := scs.availCtxs[p.availCtxI]
	ctxP := Param{
		Name:     p.name,
		IsString: p.nonBoolType == "str" || p.nonBoolType == "duration",
		IsBool:   p.isBool,
		// the Sources don't actually care about the other fields of Param,
		// those are only used by Populate once it has all ParamValues together
	}
	MustAdd(thisCtx, ctxP)
	ctxP = get(thisCtx).params[p.name] // get it back out to get any added fields

	if !p.unset {
		pv := ParamValue{Param: ctxP}
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
	gotPVs, err := s.Parse(collectParams(scs.ctx))
	if err != nil {
		return err
	}
	return massert.All(
		massert.Len(gotPVs, len(scs.expPVs)),
		massert.Subset(scs.expPVs, gotPVs),
	).Assert()
}

func TestSources(t *T) {
	ctx := mctx.New()
	a := RequiredInt(ctx, "a", "")
	b := RequiredInt(ctx, "b", "")
	c := RequiredInt(ctx, "c", "")

	err := Populate(ctx, Sources{
		SourceCLI{Args: []string{"--a=1", "--b=666"}},
		SourceEnv{Env: []string{"B=2", "C=3"}},
	})
	massert.Fatal(t, massert.All(
		massert.Nil(err),
		massert.Equal(1, *a),
		massert.Equal(2, *b),
		massert.Equal(3, *c),
	))
}

func TestSourceMap(t *T) {
	ctx := mctx.New()
	a := RequiredInt(ctx, "a", "")
	foo := mctx.ChildOf(ctx, "foo")
	b := RequiredString(foo, "b", "")
	c := Bool(foo, "c", "")

	err := Populate(ctx, SourceMap{
		"a":     "4",
		"foo-b": "bbb",
		"foo-c": "1",
	})
	massert.Fatal(t, massert.All(
		massert.Nil(err),
		massert.Equal(4, *a),
		massert.Equal("bbb", *b),
		massert.Equal(true, *c),
	))
}
