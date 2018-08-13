package mcfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
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

type testCLIState struct {
	cfg       *Cfg
	availCfgs []*Cfg

	SourceCLI
	expPVs []ParamValue
}

type testCLIApplyer struct {
	name        string
	availCfgI   int // not technically needed, but makes subsequent steps easier
	path        []string
	isBool      bool
	nonBoolType string // "int", "str", "duration", "json"
	unset       bool
	nonBoolWEq  bool // use equal sign when setting value
	nonBoolVal  string
}

func (tca testCLIApplyer) Apply(ss mtest.State) (mtest.State, error) {
	s := ss.(testCLIState)

	// the tca needs to get added to its cfg as a Param
	thisCfg := s.availCfgs[tca.availCfgI]
	p := Param{
		Name:     tca.name,
		IsString: tca.nonBoolType == "str" || tca.nonBoolType == "duration",
		IsBool:   tca.isBool,
		// the cli parser doesn't actually care about the other fields of Param,
		// those are only used by Cfg once it has all ParamValues together
	}
	thisCfg.ParamAdd(p)

	// if the arg is set then add it to the cli args and the expected output pvs
	if !tca.unset {
		arg := cliKeyPrefix
		if len(tca.path) > 0 {
			arg += strings.Join(tca.path, cliKeyJoin) + cliKeyJoin
		}
		arg += tca.name
		if !tca.isBool {
			if tca.nonBoolWEq {
				arg += "="
			} else {
				s.SourceCLI.Args = append(s.SourceCLI.Args, arg)
				arg = ""
			}
			arg += tca.nonBoolVal
		}
		s.SourceCLI.Args = append(s.SourceCLI.Args, arg)
		log.Print(strings.Join(s.SourceCLI.Args, " "))

		pv := ParamValue{
			Param: p,
			Path:  tca.path,
		}
		if tca.isBool {
			pv.Value = json.RawMessage("true")
		} else {
			switch tca.nonBoolType {
			case "str", "duration":
				pv.Value = json.RawMessage(fmt.Sprintf("%q", tca.nonBoolVal))
			case "int", "json":
				pv.Value = json.RawMessage(tca.nonBoolVal)
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
}

func TestSourceCLI(t *T) {
	chk := mtest.Checker{
		Init: func() mtest.State {
			var s testCLIState
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
		Actions: func(ss mtest.State) []mtest.Action {
			s := ss.(testCLIState)
			var tca testCLIApplyer
			if i := mrand.Intn(8); i == 0 {
				tca.name = mrand.Hex(1) + "-" + mrand.Hex(8)
			} else if i == 1 {
				tca.name = mrand.Hex(1) + "=" + mrand.Hex(8)
			} else {
				tca.name = mrand.Hex(8)
			}

			tca.availCfgI = mrand.Intn(len(s.availCfgs))
			thisCfg := s.availCfgs[tca.availCfgI]
			tca.path = thisCfg.Path

			tca.isBool = mrand.Intn(2) == 0
			if !tca.isBool {
				tca.nonBoolType = mrand.Element([]string{
					"int",
					"str",
					"duration",
					"json",
				}, nil).(string)
			}
			tca.unset = mrand.Intn(10) == 0

			if tca.isBool || tca.unset {
				return []mtest.Action{{Applyer: tca}}
			}

			tca.nonBoolWEq = mrand.Intn(2) == 0
			switch tca.nonBoolType {
			case "int":
				tca.nonBoolVal = fmt.Sprint(mrand.Int())
			case "str":
				tca.nonBoolVal = mrand.Hex(16)
			case "duration":
				dur := time.Duration(mrand.Intn(86400)) * time.Second
				tca.nonBoolVal = dur.String()
			case "json":
				b, _ := json.Marshal(map[string]int{
					mrand.Hex(4): mrand.Int(),
					mrand.Hex(4): mrand.Int(),
					mrand.Hex(4): mrand.Int(),
				})
				tca.nonBoolVal = string(b)
			}
			return []mtest.Action{{Applyer: tca}}
		},
	}

	if err := chk.RunUntil(5 * time.Second); err != nil {
		t.Fatal(err)
	}
}
