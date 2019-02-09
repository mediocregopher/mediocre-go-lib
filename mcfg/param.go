package mcfg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtime"
)

// Param is a configuration parameter which can be populated by Populate. The
// Param will exist as part of a Context, relative to its path (see the mctx
// package for more on Context path). For example, a Param with name "addr"
// under a Context with path of []string{"foo","bar"} will be setable on the CLI
// via "--foo-bar-addr". Other configuration Sources may treat the path/name
// differently, however.
//
// Param values are always unmarshaled as JSON values into the Into field of the
// Param, regardless of the actual Source.
type Param struct {
	// How the parameter will be identified within a Context.
	Name string

	// A helpful description of how a parameter is expected to be used.
	Usage string

	// If the parameter's value is expected to be read as a go string. This is
	// used for configuration sources like CLI which will automatically add
	// double-quotes around the value if they aren't already there.
	IsString bool

	// If the parameter's value is expected to be a boolean. This is used for
	// configuration sources like CLI which treat boolean parameters (aka flags)
	// differently.
	IsBool bool

	// If true then the parameter _must_ be set by at least one Source.
	Required bool

	// The pointer/interface into which the configuration value will be
	// json.Unmarshal'd. The value being pointed to also determines the default
	// value of the parameter.
	Into interface{}

	// The Context this Param was added to. NOTE that this will be automatically
	// filled in by WithParam when the Param is added to the Context.
	Context context.Context
}

func paramFullName(path []string, name string) string {
	return strings.Join(append(path, name), "-")
}

func (p Param) fuzzyParse(v string) json.RawMessage {
	if p.IsBool {
		if v == "" || v == "0" || v == "false" {
			return json.RawMessage("false")
		}
		return json.RawMessage("true")

	} else if p.IsString && (v == "" || v[0] != '"') {
		return json.RawMessage(`"` + v + `"`)
	}

	return json.RawMessage(v)
}

type ctxKey string

func getParam(ctx context.Context, name string) (Param, bool) {
	param, ok := mctx.LocalValue(ctx, ctxKey(name)).(Param)
	return param, ok
}

// WithParam returns a Context with the given Param added to it. It will panic
// if a Param with the same Name already exists in the Context.
func WithParam(ctx context.Context, param Param) context.Context {
	param.Name = strings.ToLower(param.Name)
	param.Context = ctx

	if _, ok := getParam(ctx, param.Name); ok {
		path := mctx.Path(ctx)
		panic(fmt.Sprintf("Context Path:%#v Name:%q already exists", path, param.Name))
	}

	return mctx.WithLocalValue(ctx, ctxKey(param.Name), param)
}

func getLocalParams(ctx context.Context) []Param {
	localVals := mctx.LocalValues(ctx)
	params := make([]Param, 0, len(localVals))
	for _, val := range localVals {
		if param, ok := val.(Param); ok {
			params = append(params, param)
		}
	}
	return params
}

// WithInt64 returns an *int64 which will be populated once Populate is run on
// the returned Context.
func WithInt64(ctx context.Context, name string, defaultVal int64, usage string) (context.Context, *int64) {
	i := defaultVal
	ctx = WithParam(ctx, Param{Name: name, Usage: usage, Into: &i})
	return ctx, &i
}

// WithRequiredInt64 returns an *int64 which will be populated once Populate is
// run on the returned Context, and which must be supplied by a configuration
// Source.
func WithRequiredInt64(ctx context.Context, name string, usage string) (context.Context, *int64) {
	var i int64
	ctx = WithParam(ctx, Param{Name: name, Required: true, Usage: usage, Into: &i})
	return ctx, &i
}

// WithInt returns an *int which will be populated once Populate is run on the
// returned Context.
func WithInt(ctx context.Context, name string, defaultVal int, usage string) (context.Context, *int) {
	i := defaultVal
	ctx = WithParam(ctx, Param{Name: name, Usage: usage, Into: &i})
	return ctx, &i
}

// WithRequiredInt returns an *int which will be populated once Populate is run
// on the returned Context, and which must be supplied by a configuration
// Source.
func WithRequiredInt(ctx context.Context, name string, usage string) (context.Context, *int) {
	var i int
	ctx = WithParam(ctx, Param{Name: name, Required: true, Usage: usage, Into: &i})
	return ctx, &i
}

// WithString returns a *string which will be populated once Populate is run on
// the returned Context.
func WithString(ctx context.Context, name, defaultVal, usage string) (context.Context, *string) {
	s := defaultVal
	ctx = WithParam(ctx, Param{Name: name, Usage: usage, IsString: true, Into: &s})
	return ctx, &s
}

// WithRequiredString returns a *string which will be populated once Populate is
// run on the returned Context, and which must be supplied by a configuration
// Source.
func WithRequiredString(ctx context.Context, name, usage string) (context.Context, *string) {
	var s string
	ctx = WithParam(ctx, Param{Name: name, Required: true, Usage: usage, IsString: true, Into: &s})
	return ctx, &s
}

// WithBool returns a *bool which will be populated once Populate is run on the
// returned Context, and which defaults to false if unconfigured.
//
// The default behavior of all Sources is that a boolean parameter will be set
// to true unless the value is "", 0, or false. In the case of the CLI Source
// the value will also be true when the parameter is used with no value at all,
// as would be expected.
func WithBool(ctx context.Context, name, usage string) (context.Context, *bool) {
	var b bool
	ctx = WithParam(ctx, Param{Name: name, Usage: usage, IsBool: true, Into: &b})
	return ctx, &b
}

// WithTS returns an *mtime.TS which will be populated once Populate is run on
// the returned Context.
func WithTS(ctx context.Context, name string, defaultVal mtime.TS, usage string) (context.Context, *mtime.TS) {
	t := defaultVal
	ctx = WithParam(ctx, Param{Name: name, Usage: usage, Into: &t})
	return ctx, &t
}

// WithRequiredTS returns an *mtime.TS which will be populated once Populate is
// run on the returned Context, and which must be supplied by a configuration
// Source.
func WithRequiredTS(ctx context.Context, name, usage string) (context.Context, *mtime.TS) {
	var t mtime.TS
	ctx = WithParam(ctx, Param{Name: name, Required: true, Usage: usage, Into: &t})
	return ctx, &t
}

// WithDuration returns an *mtime.Duration which will be populated once Populate
// is run on the returned Context.
func WithDuration(ctx context.Context, name string, defaultVal mtime.Duration, usage string) (context.Context, *mtime.Duration) {
	d := defaultVal
	ctx = WithParam(ctx, Param{Name: name, Usage: usage, IsString: true, Into: &d})
	return ctx, &d
}

// WithRequiredDuration returns an *mtime.Duration which will be populated once
// Populate is run on the returned Context, and which must be supplied by a
// configuration Source.
func WithRequiredDuration(ctx context.Context, name string, defaultVal mtime.Duration, usage string) (context.Context, *mtime.Duration) {
	var d mtime.Duration
	ctx = WithParam(ctx, Param{Name: name, Required: true, Usage: usage, IsString: true, Into: &d})
	return ctx, &d
}

// WithJSON reads the parameter value as a JSON value and unmarshals it into the
// given interface{} (which should be a pointer). The receiver (into) is also
// used to determine the default value.
func WithJSON(ctx context.Context, name string, into interface{}, usage string) context.Context {
	return WithParam(ctx, Param{Name: name, Usage: usage, Into: into})
}

// WithRequiredJSON reads the parameter value as a JSON value and unmarshals it
// into the given interface{} (which should be a pointer). The value must be
// supplied by a configuration Source.
func WithRequiredJSON(ctx context.Context, name string, into interface{}, usage string) context.Context {
	return WithParam(ctx, Param{Name: name, Required: true, Usage: usage, Into: into})
}
