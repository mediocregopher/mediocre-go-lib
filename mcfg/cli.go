package mcfg

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
)

type cliKey int

const (
	cliKeyTail cliKey = iota
	cliKeySubCmdM
)

type cliTail struct {
	dst   *[]string
	descr string
}

// WithCLITail returns a Context which modifies the behavior of SourceCLI's
// Parse. Normally when SourceCLI encounters an unexpected Arg it will
// immediately return an error. This function modifies the Context to indicate
// to Parse that the unexpected Arg, and all subsequent Args (i.e. the tail),
// should be set to the returned []string value.
//
// The descr (optional) will be appended to the "Usage" line which is printed
// with the help document when "-h" is passed in.
func WithCLITail(ctx context.Context, descr string) (context.Context, *[]string) {
	if ctx.Value(cliKeyTail) != nil {
		panic("WithCLITail already called in this Context")
	}
	tailPtr := new([]string)
	ctx = context.WithValue(ctx, cliKeyTail, cliTail{
		dst:   tailPtr,
		descr: descr,
	})
	return ctx, tailPtr
}

func populateCLITail(ctx context.Context, tail []string) bool {
	ct, ok := ctx.Value(cliKeyTail).(cliTail)
	if ok {
		*ct.dst = tail
	}
	return ok
}

func getCLITailDescr(ctx context.Context) string {
	ct, _ := ctx.Value(cliKeyTail).(cliTail)
	return ct.descr
}

type subCmd struct {
	name, descr string
	flag        *bool
	callback    func(context.Context) context.Context
}

// WithCLISubCommand establishes a sub-command which can be activated on the
// command-line. When a sub-command is given on the command-line, the bool
// returned for that sub-command will be set to true.
//
// Additionally, the Context which was passed into Parse (i.e. the one passed
// into Populate) will be passed into the given callback, and the returned one
// used for subsequent parsing. This allows for setting sub-command specific
// Params, sub-command specific runtime behavior (via mrun.WithStartHook),
// support for sub-sub-commands, and more. The callback may be nil.
//
// If any sub-commands have been defined on a Context which is passed into
// Parse, it is assumed that a sub-command is required on the command-line.
//
// Sub-commands must be specified before any other options on the command-line.
func WithCLISubCommand(ctx context.Context, name, descr string, callback func(context.Context) context.Context) (context.Context, *bool) {
	m, _ := ctx.Value(cliKeySubCmdM).(map[string]subCmd)
	if m == nil {
		m = map[string]subCmd{}
		ctx = context.WithValue(ctx, cliKeySubCmdM, m)
	}

	flag := new(bool)
	m[name] = subCmd{
		name:     name,
		descr:    descr,
		flag:     flag,
		callback: callback,
	}
	return ctx, flag
}

// SourceCLI is a Source which will parse configuration from the CLI.
//
// Possible CLI options are generated by joining a Param's Path and Name with
// dashes. For example:
//
//	ctx := mctx.New()
//	ctx = mctx.ChildOf(ctx, "foo")
//	ctx = mctx.ChildOf(ctx, "bar")
//	addr := mcfg.String(ctx, "addr", "", "Some address")
//	// the CLI option to fill addr will be "--foo-bar-addr"
//
// If the "-h" option is seen then a help page will be printed to
// stdout and the process will exit. Since all normally-defined parameters must
// being with double-dash ("--") they won't ever conflict with the help option.
//
// SourceCLI behaves a little differently with boolean parameters. Setting the
// value of a boolean parameter directly _must_ be done with an equals, for
// example: `--boolean-flag=1` or `--boolean-flag=false`. Using the
// space-separated format will not work. If a boolean has no equal-separated
// value it is assumed to be setting the value to `true`, as would be expected.
type SourceCLI struct {
	Args []string // if nil then os.Args[1:] is used

	DisableHelpPage bool
}

const (
	cliKeyJoin   = "-"
	cliKeyPrefix = "--"
	cliValSep    = "="
	cliHelpArg   = "-h"
)

// Parse implements the method for the Source interface
func (cli *SourceCLI) Parse(ctx context.Context) (context.Context, []ParamValue, error) {
	args := cli.Args
	if cli.Args == nil {
		args = os.Args[1:]
	}
	return cli.parse(ctx, nil, args)
}

func (cli *SourceCLI) parse(
	ctx context.Context,
	subCmdPrefix, args []string,
) (
	context.Context,
	[]ParamValue,
	error,
) {
	pM, err := cli.cliParams(CollectParams(ctx))
	if err != nil {
		return nil, nil, err
	}

	printHelpAndExit := func() {
		cli.printHelp(ctx, os.Stderr, subCmdPrefix, pM)
		os.Stderr.Sync()
		os.Exit(1)
	}

	// if sub-commands were defined on this Context then handle that first. One
	// of them should have been given, in which case send the Context through
	// the callback to obtain a new one (which presumably has further config
	// options the previous didn't) and call parse again.
	subCmdM, _ := ctx.Value(cliKeySubCmdM).(map[string]subCmd)
	if len(subCmdM) > 0 {
		subCmd, args, ok := cli.getSubCmd(subCmdM, args)
		if !ok {
			printHelpAndExit()
		}
		ctx = context.WithValue(ctx, cliKeySubCmdM, nil)
		if subCmd.callback != nil {
			ctx = subCmd.callback(ctx)
		}
		subCmdPrefix = append(subCmdPrefix, subCmd.name)
		*subCmd.flag = true
		return cli.parse(ctx, subCmdPrefix, args)
	}

	// if sub-commands were not set, then proceed with normal command-line arg
	// processing.
	pvs := make([]ParamValue, 0, len(args))
	var (
		key        string
		p          Param
		pOk        bool
		pvStrVal   string
		pvStrValOk bool
	)
	for i, arg := range args {
		if pOk {
			pvStrVal = arg
			pvStrValOk = true
		} else if !cli.DisableHelpPage && arg == cliHelpArg {
			printHelpAndExit()
		} else {
			for key, p = range pM {
				if arg == key {
					pOk = true
					break
				}

				prefix := key + cliValSep
				if !strings.HasPrefix(arg, prefix) {
					continue
				}
				pOk = true
				pvStrVal = strings.TrimPrefix(arg, prefix)
				pvStrValOk = true
				break
			}
			if !pOk {
				if ok := populateCLITail(ctx, args[i:]); ok {
					return ctx, pvs, nil
				}
				ctx := mctx.Annotate(context.Background(), "param", arg)
				return nil, nil, merr.New("unexpected config parameter", ctx)
			}
		}

		// pOk is always true at this point, and so p is filled in

		// As a special case for CLI, if a boolean has no value set it means it
		// is true.
		if p.IsBool && !pvStrValOk {
			pvStrVal = "true"
		} else if !pvStrValOk {
			// everything else should have a value. if pvStrVal isn't filled it
			// means the next arg should be one. Continue the loop, it'll get
			// filled with the next one (hopefully)
			continue
		}

		pvs = append(pvs, ParamValue{
			Name:  p.Name,
			Path:  mctx.Path(p.Context),
			Value: p.fuzzyParse(pvStrVal),
		})

		key = ""
		p = Param{}
		pOk = false
		pvStrVal = ""
		pvStrValOk = false
	}
	if pOk && !pvStrValOk {
		ctx := mctx.Annotate(p.Context, "param", key)
		return nil, nil, merr.New("param expected a value", ctx)
	}

	return ctx, pvs, nil
}

func (cli *SourceCLI) getSubCmd(subCmdM map[string]subCmd, args []string) (subCmd, []string, bool) {
	if len(args) == 0 {
		return subCmd{}, args, false
	}

	s, ok := subCmdM[args[0]]
	if !ok {
		return subCmd{}, args, false
	}

	return s, args[1:], true
}

func (cli *SourceCLI) cliParams(params []Param) (map[string]Param, error) {
	m := map[string]Param{}
	for _, p := range params {
		key := strings.Join(append(mctx.Path(p.Context), p.Name), cliKeyJoin)
		m[cliKeyPrefix+key] = p
	}
	return m, nil
}

func (cli *SourceCLI) printHelp(
	ctx context.Context,
	w io.Writer,
	subCmdPrefix []string,
	pM map[string]Param,
) {
	type pEntry struct {
		arg string
		Param
	}

	pA := make([]pEntry, 0, len(pM))
	for arg, p := range pM {
		pA = append(pA, pEntry{arg: arg, Param: p})
	}

	sort.Slice(pA, func(i, j int) bool {
		if pA[i].Required != pA[j].Required {
			return pA[i].Required
		}
		return pA[i].arg < pA[j].arg
	})

	fmtDefaultVal := func(ptr interface{}) string {
		if ptr == nil {
			return ""
		}
		val := reflect.Indirect(reflect.ValueOf(ptr))
		zero := reflect.Zero(val.Type())
		if reflect.DeepEqual(val.Interface(), zero.Interface()) {
			return ""
		} else if val.Type().Kind() == reflect.String {
			return fmt.Sprintf("%q", val.Interface())
		}
		return fmt.Sprint(val.Interface())
	}

	type subCmdEntry struct {
		name string
		subCmd
	}

	subCmdM, _ := ctx.Value(cliKeySubCmdM).(map[string]subCmd)
	subCmdA := make([]subCmdEntry, 0, len(subCmdM))
	for name, subCmd := range subCmdM {
		subCmdA = append(subCmdA, subCmdEntry{name: name, subCmd: subCmd})
	}

	sort.Slice(subCmdA, func(i, j int) bool {
		return subCmdA[i].name < subCmdA[j].name
	})

	fmt.Fprintf(w, "Usage: %s", os.Args[0])
	if len(subCmdPrefix) > 0 {
		fmt.Fprintf(w, " %s", strings.Join(subCmdPrefix, " "))
	}
	if len(subCmdA) > 0 {
		fmt.Fprint(w, " <sub-command>")
	}
	if len(pA) > 0 {
		fmt.Fprint(w, " [options]")
	}
	if descr := getCLITailDescr(ctx); descr != "" {
		fmt.Fprintf(w, " %s", descr)
	}
	fmt.Fprint(w, "\n\n")

	if len(subCmdA) > 0 {
		fmt.Fprint(w, "Sub-commands:\n\n")
		for _, subCmd := range subCmdA {
			fmt.Fprintf(w, "\t%s\t%s\n", subCmd.name, subCmd.descr)
		}
		fmt.Fprint(w, "\n")
	}

	if len(pA) > 0 {
		fmt.Fprint(w, "Options:\n\n")
		for _, p := range pA {
			fmt.Fprintf(w, "\t%s", p.arg)
			if p.IsBool {
				fmt.Fprintf(w, " (Flag)")
			} else if p.Required {
				fmt.Fprintf(w, " (Required)")
			} else if defVal := fmtDefaultVal(p.Into); defVal != "" {
				fmt.Fprintf(w, " (Default: %s)", defVal)
			}
			fmt.Fprint(w, "\n")
			if usage := p.Usage; usage != "" {
				// make all usages end with a period, because I say so
				usage = strings.TrimSpace(usage)
				if !strings.HasSuffix(usage, ".") {
					usage += "."
				}
				fmt.Fprintln(w, "\t\t"+usage)
			}
			fmt.Fprint(w, "\n")
		}
	}
}
