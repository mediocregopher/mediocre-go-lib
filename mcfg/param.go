package mcfg

import (
	"fmt"

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
}

// ParamAdd adds the given Param to the Cfg. It will panic if a Param of the
// same Name already exists in the Cfg.
func (c *Cfg) ParamAdd(p Param) {
	if _, ok := c.Params[p.Name]; ok {
		panic(fmt.Sprintf("Cfg.Path:%#v name:%q already exists", c.Path, p.Name))
	}
	c.Params[p.Name] = p
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
