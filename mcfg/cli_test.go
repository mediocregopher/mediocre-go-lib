package mcfg

import (
	"bytes"
	"context"
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest/mchk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceCLIHelp(t *T) {
	ctx := context.Background()
	ctx, _ = Int(ctx, "foo", 5, "Test int param  ") // trailing space should be trimmed
	ctx, _ = Bool(ctx, "bar", "Test bool param.")
	ctx, _ = String(ctx, "baz", "baz", "Test string param")
	ctx, _ = RequiredString(ctx, "baz2", "")
	ctx, _ = RequiredString(ctx, "baz3", "")
	src := SourceCLI{}

	buf := new(bytes.Buffer)
	pM, err := src.cliParams(collectParams(ctx))
	require.NoError(t, err)
	SourceCLI{}.printHelp(buf, pM)

	exp := `
--baz2 (Required)

--baz3 (Required)

--bar (Flag)
	Test bool param.

--baz (Default: "baz")
	Test string param.

--foo (Default: 5)
	Test int param.

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

			s.srcCommonState = s.srcCommonState.applyCtxAndPV(p.srcCommonParams)
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
