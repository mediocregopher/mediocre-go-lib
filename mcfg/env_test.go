package mcfg

import (
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mtest/mchk"
)

func TestSourceEnv(t *T) {
	type state struct {
		srcCommonState
		SourceEnv
	}

	type params struct {
		srcCommonParams
	}

	chk := mchk.Checker{
		Init: func() mchk.State {
			var s state
			s.srcCommonState = newSrcCommonState()
			s.SourceEnv.Env = make([]string, 0, 16)
			return s
		},
		Next: func(ss mchk.State) mchk.Action {
			s := ss.(state)
			var p params
			p.srcCommonParams = s.srcCommonState.next()
			return mchk.Action{Params: p}
		},
		Apply: func(ss mchk.State, a mchk.Action) (mchk.State, error) {
			s := ss.(state)
			p := a.Params.(params)
			s.srcCommonState = s.srcCommonState.applyCtxAndPV(p.srcCommonParams)
			if !p.unset {
				kv := strings.Join(append(p.path, p.name), "_")
				kv = strings.Replace(kv, "-", "_", -1)
				kv = strings.ToUpper(kv)
				kv += "="
				if p.isBool {
					kv += "1"
				} else {
					kv += p.nonBoolVal
				}
				s.SourceEnv.Env = append(s.SourceEnv.Env, kv)
			}
			err := s.srcCommonState.assert(s.SourceEnv)
			return s, err
		},
	}

	if err := chk.RunFor(2 * time.Second); err != nil {
		t.Fatal(err)
	}
}
