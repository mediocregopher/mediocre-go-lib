package mcfg

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
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
	assertHelp := func(ctx context.Context, subCmdPrefix []string, exp string) {
		buf := new(bytes.Buffer)
		src := &SourceCLI{}
		pM, err := src.cliParams(CollectParams(ctx))
		require.NoError(t, err)
		src.printHelp(ctx, buf, subCmdPrefix, pM)

		out := buf.String()
		ok := regexp.MustCompile(exp).MatchString(out)
		assert.True(t, ok, "exp:%s (%q)\ngot:%s (%q)", exp, exp, out, out)
	}

	ctx := context.Background()
	assertHelp(ctx, nil, `^Usage: \S+

$`)
	assertHelp(ctx, []string{"foo", "bar"}, `^Usage: \S+ foo bar

$`)

	ctx, _ = WithInt(ctx, "foo", 5, "Test int param  ") // trailing space should be trimmed
	ctx, _ = WithBool(ctx, "bar", "Test bool param.")
	ctx, _ = WithString(ctx, "baz", "baz", "Test string param")
	ctx, _ = WithRequiredString(ctx, "baz2", "")
	ctx, _ = WithRequiredString(ctx, "baz3", "")

	assertHelp(ctx, nil, `^Usage: \S+ \[options\]

Options:

	--baz2 \(Required\)

	--baz3 \(Required\)

	--bar \(Flag\)
		Test bool param.

	--baz \(Default: "baz"\)
		Test string param.

	--foo \(Default: 5\)
		Test int param.

$`)

	assertHelp(ctx, []string{"foo", "bar"}, `^Usage: \S+ foo bar \[options\]

Options:

	--baz2 \(Required\)

	--baz3 \(Required\)

	--bar \(Flag\)
		Test bool param.

	--baz \(Default: "baz"\)
		Test string param.

	--foo \(Default: 5\)
		Test int param.

$`)

	ctx, _ = WithCLISubCommand(ctx, "first", "First sub-command", nil)
	ctx, _ = WithCLISubCommand(ctx, "second", "Second sub-command", nil)
	assertHelp(ctx, []string{"foo", "bar"}, `^Usage: \S+ foo bar <sub-command> \[options\]

Sub-commands:

	first	First sub-command
	second	Second sub-command

Options:

	--baz2 \(Required\)

	--baz3 \(Required\)

	--bar \(Flag\)
		Test bool param.

	--baz \(Default: "baz"\)
		Test string param.

	--foo \(Default: 5\)
		Test int param.

$`)

	ctx, _ = WithCLITail(ctx, "[arg...]")
	assertHelp(ctx, nil, `^Usage: \S+ <sub-command> \[options\] \[arg\.\.\.\]

Sub-commands:

	first	First sub-command
	second	Second sub-command

Options:

	--baz2 \(Required\)

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

func TestWithCLITail(t *T) {
	ctx := context.Background()
	ctx, _ = WithInt(ctx, "foo", 5, "")
	ctx, _ = WithBool(ctx, "bar", "")

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
		ctx, tail := WithCLITail(ctx, "foo")
		_, err := Populate(ctx, &SourceCLI{Args: tc.args})
		massert.Require(t, massert.Comment(massert.All(
			massert.Nil(err),
			massert.Equal(tc.expTail, *tail),
		), "tc: %#v", tc))
	}
}

func ExampleWithCLITail() {
	ctx := context.Background()
	ctx, foo := WithInt(ctx, "foo", 1, "Description of foo.")
	ctx, tail := WithCLITail(ctx, "[arg...]")
	ctx, bar := WithString(ctx, "bar", "defaultVal", "Description of bar.")

	_, err := Populate(ctx, &SourceCLI{
		Args: []string{"--foo=100", "arg1", "arg2", "arg3"},
	})

	fmt.Printf("err:%v foo:%v bar:%v tail:%#v\n", err, *foo, *bar, *tail)
	// Output: err:<nil> foo:100 bar:defaultVal tail:[]string{"arg1", "arg2", "arg3"}
}

func TestWithCLISubCommand(t *T) {
	var (
		ctx   context.Context
		foo   *int
		bar   *int
		baz   *int
		aFlag *bool
		bFlag *bool
	)
	reset := func() {
		foo, bar, baz, aFlag, bFlag = nil, nil, nil, nil, nil
		ctx = context.Background()
		ctx, foo = WithInt(ctx, "foo", 0, "Description of foo.")
		ctx, aFlag = WithCLISubCommand(ctx, "a", "Description of a.",
			func(ctx context.Context) context.Context {
				ctx, bar = WithInt(ctx, "bar", 0, "Description of bar.")
				return ctx
			})
		ctx, bFlag = WithCLISubCommand(ctx, "b", "Description of b.",
			func(ctx context.Context) context.Context {
				ctx, baz = WithInt(ctx, "baz", 0, "Description of baz.")
				return ctx
			})
	}

	reset()
	_, err := Populate(ctx, &SourceCLI{
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
	_, err = Populate(ctx, &SourceCLI{
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

func ExampleWithCLISubCommand() {
	// Create a new Context with a parameter "foo", which can be used across all
	// sub-commands.
	ctx := context.Background()
	ctx, foo := WithInt(ctx, "foo", 0, "Description of foo.")

	// Create a sub-command "a", which has a parameter "bar" specific to it.
	var bar *int
	ctx, aFlag := WithCLISubCommand(ctx, "a", "Description of a.",
		func(ctx context.Context) context.Context {
			ctx, bar = WithInt(ctx, "bar", 0, "Description of bar.")
			return ctx
		})

	// Create a sub-command "b", which has a parameter "baz" specific to it.
	var baz *int
	ctx, bFlag := WithCLISubCommand(ctx, "b", "Description of b.",
		func(ctx context.Context) context.Context {
			ctx, baz = WithInt(ctx, "baz", 0, "Description of baz.")
			return ctx
		})

	// Use Populate with manually generated CLI arguments, calling the "a"
	// sub-command.
	args := []string{"a", "--foo=1", "--bar=2"}
	if _, err := Populate(ctx, &SourceCLI{Args: args}); err != nil {
		panic(err)
	}
	fmt.Printf("foo:%d bar:%d aFlag:%v bFlag:%v\n", *foo, *bar, *aFlag, *bFlag)

	// reset output for another Populate, this time calling the "b" sub-command.
	*aFlag = false
	args = []string{"b", "--foo=1", "--baz=3"}
	if _, err := Populate(ctx, &SourceCLI{Args: args}); err != nil {
		panic(err)
	}
	fmt.Printf("foo:%d baz:%d aFlag:%v bFlag:%v\n", *foo, *baz, *aFlag, *bFlag)

	// Output: foo:1 bar:2 aFlag:true bFlag:false
	// foo:1 baz:3 aFlag:false bFlag:true
}
