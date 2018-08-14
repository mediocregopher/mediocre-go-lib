package mcfg

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mtime"
)

// Param is a configuration parameter which can be added to a Cfg. The Param
// will exist relative to a Cfg's Path. For example, a Param with name "addr"
// under a Cfg with Path of []string{"foo","bar"} will be setabble on the CLI
// via "--foo-bar-addr". Other configuration Sources may treat the path/name
// differently, however.
type Param struct {
	// How the parameter will be identified within a Cfg instance
	Name string
	// A helpful description of how a parameter is expected to be used
	Usage string

	// If the parameter's value is expected to be read as a go string. This is
	// used for configuration sources like CLI which will automatically escape
	// the parameter's value with double-quotes
	IsString bool

	// If the parameter's value is expected to be a boolean. This is used for
	// configuration sources like CLI which treat boolean parameters (aka flags)
	// differently.
	IsBool bool

	// If true then the parameter _must_ be set by at least one configuration
	// source
	Required bool

	// The pointer/interface into which the configuration value will be
	// json.Unmarshal'd. The value being pointed to also determines the default
	// value of the parameter.
	Into interface{}

	// The Path field of the Cfg this Param is attached to. NOTE that this
	// will be automatically filled in when the Param is added to the Cfg.
	Path []string
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

func (p Param) fullName() string {
	return strings.Join(append(p.Path, p.Name), "-")
}

func (p Param) hash() string {
	h := md5.New()
	for _, path := range p.Path {
		fmt.Fprintf(h, "pathEl:%q\n", path)
	}
	fmt.Fprintf(h, "name:%q\n", p.Name)
	hStr := hex.EncodeToString(h.Sum(nil))
	// we add the displayName to it to make debugging easier
	return p.fullName() + "/" + hStr
}

// ParamAdd adds the given Param to the Cfg. It will panic if a Param of the
// same Name already exists in the Cfg.
func (c *Cfg) ParamAdd(p Param) {
	p.Name = strings.ToLower(p.Name)
	if _, ok := c.Params[p.Name]; ok {
		panic(fmt.Sprintf("Cfg.Path:%#v name:%q already exists", c.Path, p.Name))
	}
	p.Path = c.Path
	c.Params[p.Name] = p
}

func (c *Cfg) allParams() []Param {
	params := make([]Param, 0, len(c.Params))
	for _, p := range c.Params {
		params = append(params, p)
	}

	for _, child := range c.Children {
		params = append(params, child.allParams()...)
	}
	return params
}

// ParamInt64 returns an *int64 which will be populated once the Cfg is run
func (c *Cfg) ParamInt64(name string, defaultVal int64, usage string) *int64 {
	i := defaultVal
	c.ParamAdd(Param{Name: name, Usage: usage, Into: &i})
	return &i
}

// ParamRequiredInt64 returns an *int64 which will be populated once the Cfg is
// run, and which must be supplied by a configuration Source
func (c *Cfg) ParamRequiredInt64(name string, usage string) *int64 {
	var i int64
	c.ParamAdd(Param{Name: name, Required: true, Usage: usage, Into: &i})
	return &i
}

// ParamInt returns an *int which will be populated once the Cfg is run
func (c *Cfg) ParamInt(name string, defaultVal int, usage string) *int {
	i := defaultVal
	c.ParamAdd(Param{Name: name, Usage: usage, Into: &i})
	return &i
}

// ParamRequiredInt returns an *int which will be populated once the Cfg is run,
// and which must be supplied by a configuration Source
func (c *Cfg) ParamRequiredInt(name string, usage string) *int {
	var i int
	c.ParamAdd(Param{Name: name, Required: true, Usage: usage, Into: &i})
	return &i
}

// ParamString returns a *string which will be populated once the Cfg is run
func (c *Cfg) ParamString(name, defaultVal, usage string) *string {
	s := defaultVal
	c.ParamAdd(Param{Name: name, Usage: usage, IsString: true, Into: &s})
	return &s
}

// ParamRequiredString returns a *string which will be populated once the Cfg is
// run, and which must be supplied by a configuration Source
func (c *Cfg) ParamRequiredString(name, usage string) *string {
	var s string
	c.ParamAdd(Param{Name: name, Required: true, Usage: usage, IsString: true, Into: &s})
	return &s
}

// ParamBool returns a *bool which will be populated once the Cfg is run, and
// which defaults to false if unconfigured
//
// The default behavior of all Sources is that a boolean parameter will be set
// to true unless the value is "", 0, or false. In the case of the CLI Source
// the value will also be true when the parameter is used with no value at all,
// as would be expected.
func (c *Cfg) ParamBool(name, usage string) *bool {
	var b bool
	c.ParamAdd(Param{Name: name, Usage: usage, IsBool: true, Into: &b})
	return &b
}

// ParamTS returns an *mtime.TS which will be populated once the Cfg is run
func (c *Cfg) ParamTS(name string, defaultVal mtime.TS, usage string) *mtime.TS {
	t := defaultVal
	c.ParamAdd(Param{Name: name, Usage: usage, Into: &t})
	return &t
}

// ParamDuration returns an *mtime.Duration which will be populated once the Cfg
// is run
func (c *Cfg) ParamDuration(name string, defaultVal mtime.Duration, usage string) *mtime.Duration {
	d := defaultVal
	c.ParamAdd(Param{Name: name, Usage: usage, IsString: true, Into: &d})
	return &d
}

// ParamJSON reads the parameter value as a JSON value and unmarshals it into
// the given interface{} (which should be a pointer). The receiver (into) is
// also used to determine the default value.
func (c *Cfg) ParamJSON(name string, into interface{}, usage string) {
	c.ParamAdd(Param{Name: name, Usage: usage, Into: into})
}

// ParamRequiredJSON reads the parameter value as a JSON value and unmarshals it
// into the given interface{} (which should be a pointer).
func (c *Cfg) ParamRequiredJSON(name string, into interface{}, usage string) {
	c.ParamAdd(Param{Name: name, Required: true, Usage: usage, Into: into})
}
