package mcfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
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
		cfg       *Cfg
		availCfgs []*Cfg

		SourceCLI
		expPVs []ParamValue
	}

	type params struct {
		name        string
		availCfgI   int // not technically needed, but makes subsequent steps easier
		path        []string
		isBool      bool
		nonBoolType string // "int", "str", "duration", "json"
		unset       bool
		nonBoolWEq  bool // use equal sign when setting value
		nonBoolVal  string
	}

	chk := mchk.Checker{
		Init: func() mchk.State {
			var s state
			s.cfg = New()
			{
				a := s.cfg.Child("a")
				b := s.cfg.Child("b")
				c := s.cfg.Child("c")
				ab := a.Child("b")
				bc := b.Child("c")
				abc := ab.Child("c")
				s.availCfgs = []*Cfg{s.cfg, a, b, c, ab, bc, abc}
			}
			s.SourceCLI.Args = make([]string, 0, 16)
			return s
		},
		Next: func(ss mchk.State) mchk.Action {
			s := ss.(state)
			var p params
			if i := mrand.Intn(8); i == 0 {
				p.name = mrand.Hex(1) + "-" + mrand.Hex(8)
			} else if i == 1 {
				p.name = mrand.Hex(1) + "=" + mrand.Hex(8)
			} else {
				p.name = mrand.Hex(8)
			}

			p.availCfgI = mrand.Intn(len(s.availCfgs))
			thisCfg := s.availCfgs[p.availCfgI]
			p.path = thisCfg.Path

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
				return mchk.Action{Params: p}
			}

			p.nonBoolWEq = mrand.Intn(2) == 0
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
			return mchk.Action{Params: p}
		},
		Apply: func(ss mchk.State, a mchk.Action) (mchk.State, error) {
			s := ss.(state)
			p := a.Params.(params)

			// the param needs to get added to its cfg as a Param
			thisCfg := s.availCfgs[p.availCfgI]
			cfgP := Param{
				Name:     p.name,
				IsString: p.nonBoolType == "str" || p.nonBoolType == "duration",
				IsBool:   p.isBool,
				// the cli parser doesn't actually care about the other fields of Param,
				// those are only used by Cfg once it has all ParamValues together
			}
			thisCfg.ParamAdd(cfgP)

			// if the arg is set then add it to the cli args and the expected output pvs
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

				pv := ParamValue{
					Param: cfgP,
					Path:  p.path,
				}
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
				s.expPVs = append(s.expPVs, pv)
			}

			// and finally the state needs to be checked
			gotPVs, err := s.SourceCLI.Parse(s.cfg)
			if err != nil {
				return nil, err
			}
			return s, massert.All(
				massert.Len(gotPVs, len(s.expPVs)),
				massert.Subset(s.expPVs, gotPVs),
			).Assert()
		},
	}

	if err := chk.RunFor(5 * time.Second); err != nil {
		t.Fatal(err)
	}
}
