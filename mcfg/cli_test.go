package mcfg

import (
	"bytes"
	"encoding/json"
	"strconv"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// * dimension
// - dimension value
//
// * Cfg path
//   - New()
//   - New.Child("a")
//   - New.Child("a-b")
//   - New.Child("a=b")
// * Param name
//   - normal
//   - w/ "-"
//   - w/ "=" ?
// * Param type
//   - bool
//   - non-bool
//     * non-bool type
//       - int
//       - string
//         * Str value
//           - empty
//           - normal
//           - w/ -
//           - w/ =
//     * Value format
//       - w/ =
//       - w/o =

func combinate(slices ...[]string) [][]string {
	out := [][]string{{}}
	for _, slice := range slices {
		if len(slice) == 0 {
			continue
		}
		prev := out
		out = make([][]string, 0, len(prev)*len(slice))
		for _, prevSet := range prev {
			for _, sliceElem := range slice {
				prevSetCp := make([]string, len(prevSet), len(prevSet)+1)
				copy(prevSetCp, prevSet)
				prevSetCp = append(prevSetCp, sliceElem)
				out = append(out, prevSetCp)
			}
		}
	}
	return out
}

func TestCombinate(t *T) {
	type combTest struct {
		args [][]string
		exp  [][]string
	}

	tests := []combTest{
		{
			args: [][]string{},
			exp:  [][]string{{}},
		},
		{
			args: [][]string{{"a"}},
			exp:  [][]string{{"a"}},
		},
		{
			args: [][]string{{"a"}, {"b"}},
			exp:  [][]string{{"a", "b"}},
		},
		{
			args: [][]string{{"a", "aa"}, {"b"}},
			exp: [][]string{
				{"a", "b"},
				{"aa", "b"},
			},
		},
		{
			args: [][]string{{"a"}, {"b", "bb"}},
			exp: [][]string{
				{"a", "b"},
				{"a", "bb"},
			},
		},
		{
			args: [][]string{{"a", "aa"}, {"b", "bb"}},
			exp: [][]string{
				{"a", "b"},
				{"aa", "b"},
				{"a", "bb"},
				{"aa", "bb"},
			},
		},
	}

	for i, test := range tests {
		msgAndArgs := []interface{}{"test:%d args:%v", i, test.args}
		got := combinate(test.args...)
		assert.Len(t, got, len(test.exp), msgAndArgs...)
		for _, expHas := range test.exp {
			assert.Contains(t, got, expHas, msgAndArgs...)
		}
	}
}

func TestSourceCLI(t *T) {
	var (
		paths = []string{
			"root",
			"child",
			"childDash",
			"childEq",
		}

		paramNames = []string{
			"normal",
			"wDash",
			"wEq",
		}

		isBool = []string{
			"isBool",
			"isNotBool",
		}

		nonBoolTypes = []string{
			"int",
			"str",
		}

		nonBoolFmts = []string{
			"wEq",
			"woEq",
		}

		nonBoolStrValues = []string{
			"empty",
			"normal",
			"wDash",
			"wEq",
		}
	)

	type cliTest struct {
		path            string
		name            string
		isBool          bool
		nonBoolType     string
		nonBoolStrValue string
		nonBoolFmt      string

		// it's kinda hacky to make this a pointer, but it makes the code a lot
		// easier to read later
		exp *ParamValue
	}

	var tests []cliTest
	for _, comb := range combinate(paths, paramNames, isBool) {
		var test cliTest
		test.path = comb[0]
		test.name = comb[1]
		test.isBool = comb[2] == "isBool"
		if test.isBool {
			tests = append(tests, test)
			continue
		}

		for _, nonBoolComb := range combinate(nonBoolTypes, nonBoolFmts) {
			test.nonBoolType = nonBoolComb[0]
			test.nonBoolFmt = nonBoolComb[1]
			if test.nonBoolType != "str" {
				tests = append(tests, test)
				continue
			}
			for _, nonBoolStrValue := range nonBoolStrValues {
				test.nonBoolStrValue = nonBoolStrValue
				tests = append(tests, test)
			}
		}
	}

	childName := mrand.Hex(8)
	childDashName := mrand.Hex(4) + "-" + mrand.Hex(4)
	childEqName := mrand.Hex(4) + "=" + mrand.Hex(4)

	var args []string
	rootCfg := New()
	childCfg := rootCfg.Child(childName)
	childDashCfg := rootCfg.Child(childDashName)
	childEqCfg := rootCfg.Child(childEqName)

	for i := range tests {
		var pv ParamValue
		tests[i].exp = &pv

		switch tests[i].name {
		case "normal":
			pv.Name = mrand.Hex(8)
		case "wDash":
			pv.Name = mrand.Hex(4) + "-" + mrand.Hex(4)
		case "wEq":
			pv.Name = mrand.Hex(4) + "=" + mrand.Hex(4)
		}

		pv.IsBool = tests[i].isBool
		pv.IsString = !tests[i].isBool && tests[i].nonBoolType == "str"

		var arg string
		switch tests[i].path {
		case "root":
			rootCfg.ParamAdd(pv.Param)
			arg = "--" + pv.Name
		case "child":
			childCfg.ParamAdd(pv.Param)
			pv.Path = append(pv.Path, childName)
			arg = "--" + childName + "-" + pv.Name
		case "childDash":
			childDashCfg.ParamAdd(pv.Param)
			pv.Path = append(pv.Path, childDashName)
			arg = "--" + childDashName + "-" + pv.Name
		case "childEq":
			childEqCfg.ParamAdd(pv.Param)
			pv.Path = append(pv.Path, childEqName)
			arg = "--" + childEqName + "-" + pv.Name
		}

		if pv.IsBool {
			pv.Value = json.RawMessage("true")
			args = append(args, arg)
			continue
		}

		var val string
		switch tests[i].nonBoolType {
		case "int":
			val = strconv.Itoa(mrand.Int())
			pv.Value = json.RawMessage(val)
		case "str":
			switch tests[i].nonBoolStrValue {
			case "empty":
				// ez
			case "normal":
				val = mrand.Hex(8)
			case "wDash":
				val = mrand.Hex(4) + "-" + mrand.Hex(4)
			case "wEq":
				val = mrand.Hex(4) + "=" + mrand.Hex(4)
			}
			pv.Value = json.RawMessage(`"` + val + `"`)
		}

		switch tests[i].nonBoolFmt {
		case "wEq":
			arg += "=" + val
			args = append(args, arg)
		case "woEq":
			args = append(args, arg, val)
		}
	}

	src := SourceCLI{Args: args}
	pvals, err := src.Parse(rootCfg)
	require.NoError(t, err)
	assert.Len(t, pvals, len(tests))

	for _, test := range tests {
		assert.Contains(t, pvals, *test.exp)
	}

	// an extra bogus param or value should generate an error
	src = SourceCLI{Args: append(args, "foo")}
	_, err = src.Parse(rootCfg)
	require.Error(t, err)

}

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
