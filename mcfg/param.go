package mcfg

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtime"
)

// Param is a configuration parameter which can be populated by Populate. The
// Param will exist as part of an mctx.Context, relative to its Path. For
// example, a Param with name "addr" under an mctx.Context with Path of
// []string{"foo","bar"} will be setabble on the CLI via "--foo-bar-addr". Other
// configuration Sources may treat the path/name differently, however.
//
// Param values are always unmarshaled as JSON values into the Into field of the
// Param, regardless of the actual Source.
type Param struct {
	// How the parameter will be identified within an mctx.Context.
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

	// The Path field of the Cfg this Param is attached to. NOTE that this
	// will be automatically filled in when the Param is added to the Cfg.
	Path []string
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

// MustAdd adds the given Param to the mctx.Context. It will panic if a Param of
// the same Name already exists in the mctx.Context.
func MustAdd(ctx mctx.Context, param Param) {
	param.Name = strings.ToLower(param.Name)
	param.Path = mctx.Path(ctx)

	cfg := get(ctx)
	if _, ok := cfg.params[param.Name]; ok {
		panic(fmt.Sprintf("Context Path:%#v Name:%q already exists", param.Path, param.Name))
	}
	cfg.params[param.Name] = param
}

// Int64 returns an *int64 which will be populated once Populate is run.
func Int64(ctx mctx.Context, name string, defaultVal int64, usage string) *int64 {
	i := defaultVal
	MustAdd(ctx, Param{Name: name, Usage: usage, Into: &i})
	return &i
}

// RequiredInt64 returns an *int64 which will be populated once Populate is run,
// and which must be supplied by a configuration Source.
func RequiredInt64(ctx mctx.Context, name string, usage string) *int64 {
	var i int64
	MustAdd(ctx, Param{Name: name, Required: true, Usage: usage, Into: &i})
	return &i
}

// Int returns an *int which will be populated once Populate is run.
func Int(ctx mctx.Context, name string, defaultVal int, usage string) *int {
	i := defaultVal
	MustAdd(ctx, Param{Name: name, Usage: usage, Into: &i})
	return &i
}

// RequiredInt returns an *int which will be populated once Populate is run, and
// which must be supplied by a configuration Source.
func RequiredInt(ctx mctx.Context, name string, usage string) *int {
	var i int
	MustAdd(ctx, Param{Name: name, Required: true, Usage: usage, Into: &i})
	return &i
}

// String returns a *string which will be populated once Populate is run.
func String(ctx mctx.Context, name, defaultVal, usage string) *string {
	s := defaultVal
	MustAdd(ctx, Param{Name: name, Usage: usage, IsString: true, Into: &s})
	return &s
}

// RequiredString returns a *string which will be populated once Populate is
// run, and which must be supplied by a configuration Source.
func RequiredString(ctx mctx.Context, name, usage string) *string {
	var s string
	MustAdd(ctx, Param{Name: name, Required: true, Usage: usage, IsString: true, Into: &s})
	return &s
}

// Bool returns a *bool which will be populated once Populate is run, and which
// defaults to false if unconfigured.
//
// The default behavior of all Sources is that a boolean parameter will be set
// to true unless the value is "", 0, or false. In the case of the CLI Source
// the value will also be true when the parameter is used with no value at all,
// as would be expected.
func Bool(ctx mctx.Context, name, usage string) *bool {
	var b bool
	MustAdd(ctx, Param{Name: name, Usage: usage, IsBool: true, Into: &b})
	return &b
}

// TS returns an *mtime.TS which will be populated once Populate is run.
func TS(ctx mctx.Context, name string, defaultVal mtime.TS, usage string) *mtime.TS {
	t := defaultVal
	MustAdd(ctx, Param{Name: name, Usage: usage, Into: &t})
	return &t
}

// RequiredTS returns an *mtime.TS which will be populated once Populate is run,
// and which must be supplied by a configuration Source.
func RequiredTS(ctx mctx.Context, name, usage string) *mtime.TS {
	var t mtime.TS
	MustAdd(ctx, Param{Name: name, Required: true, Usage: usage, Into: &t})
	return &t
}

// Duration returns an *mtime.Duration which will be populated once
// Populate is run.
func Duration(ctx mctx.Context, name string, defaultVal mtime.Duration, usage string) *mtime.Duration {
	d := defaultVal
	MustAdd(ctx, Param{Name: name, Usage: usage, IsString: true, Into: &d})
	return &d
}

// RequiredDuration returns an *mtime.Duration which will be populated once
// Populate is run, and which must be supplied by a configuration Source.
func RequiredDuration(ctx mctx.Context, name string, defaultVal mtime.Duration, usage string) *mtime.Duration {
	var d mtime.Duration
	MustAdd(ctx, Param{Name: name, Required: true, Usage: usage, IsString: true, Into: &d})
	return &d
}

// JSON reads the parameter value as a JSON value and unmarshals it into the
// given interface{} (which should be a pointer). The receiver (into) is also
// used to determine the default value.
func JSON(ctx mctx.Context, name string, into interface{}, usage string) {
	MustAdd(ctx, Param{Name: name, Usage: usage, Into: into})
}

// RequiredJSON reads the parameter value as a JSON value and unmarshals it into
// the given interface{} (which should be a pointer). The value must be supplied
// by a configuration Source.
func RequiredJSON(ctx mctx.Context, name string, into interface{}, usage string) {
	MustAdd(ctx, Param{Name: name, Required: true, Usage: usage, Into: into})
}
