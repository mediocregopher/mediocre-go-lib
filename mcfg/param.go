package mcfg

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mtime"
)

// Param is a configuration parameter which can be populated by Populate. The
// Param will exist as part of a Component. For example, a Param with name
// "addr" under a Component with path of []string{"foo","bar"} will be setable
// on the CLI via "--foo-bar-addr". Other configuration Sources may treat the
// path/name differently, however.
//
// Param values are always unmarshaled as JSON values into the Into field of the
// Param, regardless of the actual Source.
type Param struct {
	// How the parameter will be identified within a Component.
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

	// The Component this Param was added to. NOTE that this will be
	// automatically filled in by AddParam when the Param is added to the
	// Component.
	Component *mcmp.Component
}

// ParamOption is a modifier which can be passed into most Param-generating
// functions (e.g. String, Int, etc...)
type ParamOption func(*Param)

// ParamRequired returns a ParamOption which ensures the parameter is required
// to be set by some configuration source. The default value of the parameter
// will be ignored.
func ParamRequired() ParamOption {
	return func(param *Param) {
		param.Required = true
	}
}

// ParamDefault returns a ParamOption which ensures the parameter uses the given
// default value when no Sources set a value for it. If not given then mcfg will
// use the zero value of the Param's type as the default value.
//
// If ParamRequired is given then this does nothing.
func ParamDefault(value interface{}) ParamOption {
	return func(param *Param) {
		intoV := reflect.ValueOf(param.Into).Elem()
		valueV := reflect.ValueOf(value)

		intoType, valueType := intoV.Type(), valueV.Type()
		if intoType != valueType {
			panic(fmt.Sprintf("ParamDefault value is type %s, but should be %s", valueType, intoType))
		} else if !intoV.CanSet() {
			panic(fmt.Sprintf("Param.Into value %#v can't be set using reflection", param.Into))
		}

		intoV.Set(valueV)
	}
}

// ParamDefaultOrRequired returns a ParamOption whose behavior depends on the
// given value. If the given value is the zero value for its type, then this returns
// ParamRequired(), otherwise this returns ParamDefault(value).
func ParamDefaultOrRequired(value interface{}) ParamOption {
	v := reflect.ValueOf(value)
	zero := reflect.Zero(v.Type())
	if v.Interface() == zero.Interface() {
		return ParamRequired()
	}
	return ParamDefault(value)
}

// ParamUsage returns a ParamOption which sets the usage string on the Param.
// This is used in some Sources, like SourceCLI, when displaying information
// about available parameters.
func ParamUsage(usage string) ParamOption {
	// make all usages end with a period, because I say so
	usage = strings.TrimSpace(usage)
	if !strings.HasSuffix(usage, ".") {
		usage += "."
	}

	return func(param *Param) {
		param.Usage = usage
	}
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

type cmpParamKey string

// used in tests
func getParam(cmp *mcmp.Component, name string) (Param, bool) {
	param, ok := cmp.Value(cmpParamKey(name)).(Param)
	return param, ok
}

// AddParam adds the given Param to the given Component. It will panic if a
// Param with the same Name already exists in the Component.
func AddParam(cmp *mcmp.Component, param Param, opts ...ParamOption) {
	param.Name = strings.ToLower(param.Name)
	param.Component = cmp
	key := cmpParamKey(param.Name)

	if cmp.HasValue(key) {
		path := cmp.Path()
		panic(fmt.Sprintf("Component.Path:%#v Param.Name:%q already exists", path, param.Name))
	}

	for _, opt := range opts {
		opt(&param)
	}
	cmp.SetValue(key, param)
}

func getLocalParams(cmp *mcmp.Component) []Param {
	values := cmp.Values()
	params := make([]Param, 0, len(values))
	for _, val := range values {
		if param, ok := val.(Param); ok {
			params = append(params, param)
		}
	}
	return params
}

// Int64 returns an *int64 which will be populated once Populate is run on the
// Component.
func Int64(cmp *mcmp.Component, name string, opts ...ParamOption) *int64 {
	var i int64
	AddParam(cmp, Param{Name: name, Into: &i}, opts...)
	return &i
}

// Int returns an *int which will be populated once Populate is run on the
// Component.
func Int(cmp *mcmp.Component, name string, opts ...ParamOption) *int {
	var i int
	AddParam(cmp, Param{Name: name, Into: &i}, opts...)
	return &i
}

// Float64 returns a *float64 which will be populated once Populate is run on
// the Component
func Float64(cmp *mcmp.Component, name string, opts ...ParamOption) *float64 {
	var f float64
	AddParam(cmp, Param{Name: name, Into: &f}, opts...)
	return &f
}

// String returns a *string which will be populated once Populate is run on
// the Component.
func String(cmp *mcmp.Component, name string, opts ...ParamOption) *string {
	var s string
	AddParam(cmp, Param{Name: name, IsString: true, Into: &s}, opts...)
	return &s
}

// Bool returns a *bool which will be populated once Populate is run on the
// Component, and which defaults to false if unconfigured.
//
// The default behavior of all Sources is that a boolean parameter will be set
// to true unless the value is "", 0, or false. In the case of the CLI Source
// the value will also be true when the parameter is used with no value at all,
// as would be expected.
func Bool(cmp *mcmp.Component, name string, opts ...ParamOption) *bool {
	var b bool
	AddParam(cmp, Param{Name: name, IsBool: true, Into: &b}, opts...)
	return &b
}

// TS returns an *mtime.TS which will be populated once Populate is run on
// the Component.
func TS(cmp *mcmp.Component, name string, opts ...ParamOption) *mtime.TS {
	var t mtime.TS
	AddParam(cmp, Param{Name: name, Into: &t}, opts...)
	return &t
}

// Duration returns an *mtime.Duration which will be populated once Populate
// is run on the Component.
func Duration(cmp *mcmp.Component, name string, opts ...ParamOption) *mtime.Duration {
	var d mtime.Duration
	AddParam(cmp, Param{Name: name, IsString: true, Into: &d}, opts...)
	return &d
}

// JSON reads the parameter value as a JSON value and unmarshals it into the
// given interface{} (which should be a pointer) once Populate is run on the
// Component.
//
// The receiver (into) is also used to determine the default value. ParamDefault
// should not be used as one of the opts.
func JSON(cmp *mcmp.Component, name string, into interface{}, opts ...ParamOption) {
	AddParam(cmp, Param{Name: name, Into: into}, opts...)
}
