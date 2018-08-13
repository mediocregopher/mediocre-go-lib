package mcfg

import (
	"bytes"
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest/mchk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceCLIHelp(t *T) {
	cfg := New()
	cfg.ParamInt("foo", 5, "Test int param")
	cfg.ParamBool("bar", "Test bool param")
	cfg.ParamString("baz", "baz", "Test string param")
	cfg.ParamString("baz2", "", "")
	src := SourceCLI{}

	buf := new(bytes.Buffer)
	pvM, err := src.cliParamVals(cfg)
	require.NoError(t, err)
	SourceCLI{}.printHelp(buf, pvM)

	exp := `
--bar (Flag)
	Test bool param

--baz (Default: "baz")
	Test string param

--baz2

--foo (Default: 5)
	Test int param

`
	assert.Equal(t, exp, buf.String())
}

func TestSourceCLI(t *T) {
	type state struct {
		srcCommonState
		SourceCLI
	}

	type params struct {
		srcCommonParams
		nonBoolWEq bool // use equal sign when setting value
	}

	chk := mchk.Checker{
		Init: func() mchk.State {
			var s state
			s.srcCommonState = newSrcCommonState()
			s.SourceCLI.Args = make([]string, 0, 16)
			return s
		},
		Next: func(ss mchk.State) mchk.Action {
			s := ss.(state)
			var p params
			p.srcCommonParams = s.srcCommonState.next()
			// if the param is a bool or unset this won't get used, but w/e
			p.nonBoolWEq = mrand.Intn(2) == 0
			return mchk.Action{Params: p}
		},
		Apply: func(ss mchk.State, a mchk.Action) (mchk.State, error) {
			s := ss.(state)
			p := a.Params.(params)

			s.srcCommonState = s.srcCommonState.applyCfgAndPV(p.srcCommonParams)
			if !p.unset {
				arg := cliKeyPrefix
				if len(p.path) > 0 {
					arg += strings.Join(p.path, cliKeyJoin) + cliKeyJoin
				}
				arg += p.name
				if !p.isBool {
					if p.nonBoolWEq {
						arg += "="
					} else {
						s.SourceCLI.Args = append(s.SourceCLI.Args, arg)
						arg = ""
					}
					arg += p.nonBoolVal
				}
				s.SourceCLI.Args = append(s.SourceCLI.Args, arg)
			}

			err := s.srcCommonState.assert(s.SourceCLI)
			return s, err
		},
	}

	if err := chk.RunFor(2 * time.Second); err != nil {
		t.Fatal(err)
	}
}
