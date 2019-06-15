package mcfg

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
	"github.com/mediocregopher/mediocre-go-lib/mtest/mchk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceCLIHelp(t *T) {
	assertHelp := func(cmp *mcmp.Component, subCmdPrefix []string, exp string) {
		buf := new(bytes.Buffer)
		src := &SourceCLI{}
		pM, err := src.cliParams(CollectParams(cmp))
		require.NoError(t, err)
		src.printHelp(cmp, buf, subCmdPrefix, pM)

		out := buf.String()
		ok := regexp.MustCompile(exp).MatchString(out)
		assert.True(t, ok, "exp:%s (%q)\ngot:%s (%q)", exp, exp, out, out)
	}

	cmp := new(mcmp.Component)
	assertHelp(cmp, nil, `^Usage: \S+

$`)
	assertHelp(cmp, []string{"foo", "bar"}, `^Usage: \S+ foo bar

$`)

	Int(cmp, "foo", ParamDefault(5), ParamUsage("Test int param  ")) // trailing space should be trimmed
	Bool(cmp, "bar", ParamUsage("Test bool param."))
	String(cmp, "baz", ParamDefault("baz"), ParamUsage("Test string param"))
	String(cmp, "baz2", ParamUsage("Required string param"), ParamRequired())
	String(cmp, "baz3", ParamRequired())

	assertHelp(cmp, nil, `^Usage: \S+ \[options\]

Options:

	--baz2 \(Required\)
		Required string param.

	--baz3 \(Required\)

	--bar \(Flag\)
		Test bool param.

	--baz \(Default: "baz"\)
		Test string param.

	--foo \(Default: 5\)
		Test int param.

$`)

	assertHelp(cmp, []string{"foo", "bar"}, `^Usage: \S+ foo bar \[options\]

Options:

	--baz2 \(Required\)
		Required string param.

	--baz3 \(Required\)

	--bar \(Flag\)
		Test bool param.

	--baz \(Default: "baz"\)
		Test string param.

	--foo \(Default: 5\)
		Test int param.

$`)

	CLISubCommand(cmp, "first", "First sub-command", nil)
	CLISubCommand(cmp, "second", "Second sub-command", nil)
	assertHelp(cmp, []string{"foo", "bar"}, `^Usage: \S+ foo bar <sub-command> \[options\]

Sub-commands:

	first	First sub-command
	second	Second sub-command

Options:

	--baz2 \(Required\)
		Required string param.

	--baz3 \(Required\)

	--bar \(Flag\)
		Test bool param.

	--baz \(Default: "baz"\)
		Test string param.

	--foo \(Default: 5\)
		Test int param.

$`)

	CLITail(cmp, "[arg...]")
	assertHelp(cmp, nil, `^Usage: \S+ <sub-command> \[options\] \[arg\.\.\.\]

Sub-commands:

	first	First sub-command
	second	Second sub-command

Options:

	--baz2 \(Required\)
		Required string param.

	--baz3 \(Required\)

	--bar \(Flag\)
		Test bool param.

	--baz \(Default: "baz"\)
		Test string param.

	--foo \(Default: 5\)
		Test int param.

$`)
}

func TestSourceCLI(t *T) {
	type state struct {
		srcCommonState
		*SourceCLI
	}

	type params struct {
		srcCommonParams
		nonBoolWEq bool // use equal sign when setting value
	}

	chk := mchk.Checker{
		Init: func() mchk.State {
			var s state
			s.srcCommonState = newSrcCommonState()
			s.SourceCLI = &SourceCLI{
				Args: make([]string, 0, 16),
			}
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

			s.srcCommonState = s.srcCommonState.applyCmpAndPV(p.srcCommonParams)
			if !p.unset {
				arg := cliKeyPrefix
				if path := p.cmp.Path(); len(path) > 0 {
					arg += strings.Join(path, cliKeyJoin) + cliKeyJoin
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

func TestCLITail(t *T) {
	cmp := new(mcmp.Component)
	Int(cmp, "foo", ParamDefault(5))
	Bool(cmp, "bar")

	type testCase struct {
		args    []string
		expTail []string
	}

	cases := []testCase{
		{
			args:    []string{"--foo", "5"},
			expTail: nil,
		},
		{
			args:    []string{"--foo", "5", "a", "b", "c"},
			expTail: []string{"a", "b", "c"},
		},
		{
			args:    []string{"--foo=5", "a", "b", "c"},
			expTail: []string{"a", "b", "c"},
		},
		{
			args:    []string{"--foo", "5", "--bar"},
			expTail: nil,
		},
		{
			args:    []string{"--foo", "5", "--bar", "a", "b", "c"},
			expTail: []string{"a", "b", "c"},
		},
	}

	for _, tc := range cases {
		tail := CLITail(cmp, "foo")
		err := Populate(cmp, &SourceCLI{Args: tc.args})
		massert.Require(t, massert.Comment(massert.All(
			massert.Nil(err),
			massert.Equal(tc.expTail, *tail),
		), "tc: %#v", tc))
	}
}

func ExampleCLITail() {
	cmp := new(mcmp.Component)
	foo := Int(cmp, "foo", ParamDefault(1), ParamUsage("Description of foo."))
	tail := CLITail(cmp, "[arg...]")
	bar := String(cmp, "bar", ParamDefault("defaultVal"), ParamUsage("Description of bar."))

	err := Populate(cmp, &SourceCLI{
		Args: []string{"--foo=100", "arg1", "arg2", "arg3"},
	})

	fmt.Printf("err:%v foo:%v bar:%v tail:%#v\n", err, *foo, *bar, *tail)
	// Output: err:<nil> foo:100 bar:defaultVal tail:[]string{"arg1", "arg2", "arg3"}
}

func TestCLISubCommand(t *T) {
	var (
		cmp   *mcmp.Component
		foo   *int
		bar   *int
		baz   *int
		aFlag *bool
		bFlag *bool
	)
	reset := func() {
		foo, bar, baz, aFlag, bFlag = nil, nil, nil, nil, nil
		cmp = new(mcmp.Component)
		foo = Int(cmp, "foo")
		aFlag = CLISubCommand(cmp, "a", "Description of a.",
			func(cmp *mcmp.Component) {
				bar = Int(cmp, "bar")
			})
		bFlag = CLISubCommand(cmp, "b", "Description of b.",
			func(cmp *mcmp.Component) {
				baz = Int(cmp, "baz")
			})
	}

	reset()
	err := Populate(cmp, &SourceCLI{
		Args: []string{"a", "--foo=1", "--bar=2"},
	})
	massert.Require(t,
		massert.Comment(massert.Nil(err), "%v", err),
		massert.Equal(1, *foo),
		massert.Equal(2, *bar),
		massert.Nil(baz),
		massert.Equal(true, *aFlag),
		massert.Equal(false, *bFlag),
	)

	reset()
	err = Populate(cmp, &SourceCLI{
		Args: []string{"b", "--foo=1", "--baz=3"},
	})
	massert.Require(t,
		massert.Comment(massert.Nil(err), "%v", err),
		massert.Equal(1, *foo),
		massert.Nil(bar),
		massert.Equal(3, *baz),
		massert.Equal(false, *aFlag),
		massert.Equal(true, *bFlag),
	)
}

func ExampleCLISubCommand() {
	var (
		cmp           *mcmp.Component
		foo, bar, baz *int
		aFlag, bFlag  *bool
	)

	// resetExample re-initializes all variables used in this example. We'll
	// call it multiple times to show different behaviors depending on what
	// arguments are passed in.
	resetExample := func() {
		// Create a new Component with a parameter "foo", which can be used across
		// all sub-commands.
		cmp = new(mcmp.Component)
		foo = Int(cmp, "foo")

		// Create a sub-command "a", which has a parameter "bar" specific to it.
		aFlag = CLISubCommand(cmp, "a", "Description of a.",
			func(cmp *mcmp.Component) {
				bar = Int(cmp, "bar")
			})

		// Create a sub-command "b", which has a parameter "baz" specific to it.
		bFlag = CLISubCommand(cmp, "b", "Description of b.",
			func(cmp *mcmp.Component) {
				baz = Int(cmp, "baz")
			})
	}

	// Use Populate with manually generated CLI arguments, calling the "a"
	// sub-command.
	resetExample()
	args := []string{"a", "--foo=1", "--bar=2"}
	if err := Populate(cmp, &SourceCLI{Args: args}); err != nil {
		panic(err)
	}
	fmt.Printf("foo:%d bar:%d aFlag:%v bFlag:%v\n", *foo, *bar, *aFlag, *bFlag)

	// reset for another Populate, this time calling the "b" sub-command.
	resetExample()
	args = []string{"b", "--foo=1", "--baz=3"}
	if err := Populate(cmp, &SourceCLI{Args: args}); err != nil {
		panic(err)
	}
	fmt.Printf("foo:%d baz:%d aFlag:%v bFlag:%v\n", *foo, *baz, *aFlag, *bFlag)

	// Output: foo:1 bar:2 aFlag:true bFlag:false
	// foo:1 baz:3 aFlag:false bFlag:true
}
