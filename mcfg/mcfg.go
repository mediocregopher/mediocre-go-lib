// Package mcfg provides a simple foundation for complex service/binary
// configuration, initialization, and destruction
package mcfg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// TODO Sources:
// - Env
// - Env file
// - JSON file
// - YAML file

// Hook describes a function which can have other Hook functions appended to it
// via the Then method. A Hook is expected to return context.Canceled on context
// cancellation.
type Hook func(context.Context) error

// Nop returns a Hook which does nothing
func Nop() Hook {
	return func(context.Context) error { return nil }
}

// Then modifies the called upon Hook such that it will first perform whatever
// it's original functionality was, and then if that doesn't return an error it
// will subsequently perform the given Hook.
func (h *Hook) Then(h2 Hook) {
	oldh := *h
	*h = func(ctx context.Context) error {
		if err := oldh(ctx); err != nil {
			return err
		} else if err := ctx.Err(); err != nil {
			// in case the previous hook doesn't respect the context
			return err
		}
		return h2(ctx)
	}
}

// Also modifies the called upon Hook such that it will perform the original
// functionality at the same time as the given Hook, wait for both to complete,
// and return an error if there is one.
func (h *Hook) Also(h2 Hook) {
	oldh := *h
	*h = func(ctx context.Context) error {
		errCh := make(chan error, 1)
		go func() {
			errCh <- oldh(ctx)
		}()
		err := h2(ctx) // don't immediately return this, wait for h2 or ctx
		if err := ctx.Err(); err != nil {
			// in case the previous hook doesn't respect the context
			return err
		} else if err := <-errCh; err != nil {
			return err
		}
		return err
	}
}

// Cfg describes a set of configuration parameters and run-time behaviors.
// Parameters are defined using the Param* methods, and run-time behaviors by
// the Hook fields on this struct.
//
// Each Cfg can have child Cfg's spawned off of it using the Child method, which
// allows for namespacing related params/behaviors into heirarchies.
type Cfg struct {
	// Read-only. The set of names passed into Child methods used to generate
	// this Cfg and all of its parents. Path will be nil if this Cfg was created
	// with New and not a Child call.
	//
	// Examples:
	// New().Path                           is nil
	// New().Child("foo").Path              is []string{"foo"}
	// New().Child("foo").Child("bar").Path is []string{"foo", "bar"}
	Path []string

	// Read-only. The set of children spawned off of this Cfg via the Child
	// method, keyed by the children's names.
	Children map[string]*Cfg

	// Read-only. The set of Params which have been added to the Cfg instance
	// via its Add method.
	Params map[string]Param

	// Start hook is performed after configuration variables have been parsed
	// and populated. All Start hooks are expected to run in a finite amount of
	// time, any long running processes spun off from them should do so in a
	// separate go-routine
	Start Hook

	// Stop hook is performed on interrupt signal, and should stop all
	// go-routines and close all resource handlers created during Start
	Stop Hook
}

// New initializes and returns an empty Cfg with default values filled in
func New() *Cfg {
	return &Cfg{
		Children: map[string]*Cfg{},
		Params:   map[string]Param{},
		Start:    Nop(),
		Stop:     Nop(),
	}
}

// IsRoot returns true if this is the root instance of a Cfg (i.e. the one
// returned by New)
func (c *Cfg) IsRoot() bool {
	return len(c.Path) == 0
}

// Name returns the name given to this instance when it was created via Child.
// if this instance was created via New (i.e. it is the root instance) then
// empty string is returned.
func (c *Cfg) Name() string {
	if c.IsRoot() {
		return ""
	}
	return c.Path[len(c.Path)-1]
}

// FullName returns a string representing the full path of the instance.
func (c *Cfg) FullName() string {
	return "/" + strings.Join(c.Path, "/")
}

func (c *Cfg) populateParams(src Source) error {
	// we allow for nil Source here for tests
	// TODO make Source stub type which tests could use here instead
	var pvs []ParamValue
	if src != nil {
		var err error
		if pvs, err = src.Parse(c); err != nil {
			return err
		}
	}

	// dedupe the ParamValues based on their hashes, with the last ParamValue
	// taking precedence
	pvM := map[string]ParamValue{}
	for _, pv := range pvs {
		pvM[pv.hash()] = pv
	}

	// check for required params
	for _, pv := range c.allParamValues() {
		if !pv.Param.Required {
			continue
		} else if _, ok := pvM[pv.hash()]; !ok {
			return fmt.Errorf("param %q is required", pv.displayName())
		}
	}

	for _, pv := range pvM {
		if err := json.Unmarshal(pv.Value, pv.Into); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cfg) runPreBlock(ctx context.Context, src Source) error {
	if err := c.populateParams(src); err != nil {
		return err
	}

	return c.Start(ctx)
}

// StartRun blocks while performing all steps of a Cfg run. The steps, in order,
// are:
// * Populate all configuration parameters
// * Perform Start hooks
//
// If any step returns an error then everything returns that error immediately.
func (c *Cfg) StartRun(ctx context.Context, src Source) error {
	return c.runPreBlock(ctx, src)
}

// StartTestRun is like StartRun, except it's intended to only be used during
// tests to initialize other entities which are going to actually be tested. It
// assumes all default configuration param values, and will return after the
// Start hook has completed. It will panic on any errors.
func (c *Cfg) StartTestRun() {
	if err := c.runPreBlock(context.Background(), nil); err != nil {
		panic(err)
	}
}

// StopRun blocks while calling the Stop hook of the Cfg, returning any error
// that it does.
func (c *Cfg) StopRun(ctx context.Context) error {
	return c.Stop(ctx)
}

// Child returns a sub-Cfg of the callee with the given name. The name will be
// prepended to all configuration options created in the returned sub-Cfg, and
// must not be empty.
func (c *Cfg) Child(name string) *Cfg {
	name = strings.ToLower(name)
	if _, ok := c.Children[name]; ok {
		panic(fmt.Sprintf("child Cfg named %q already exists", name))
	}
	c2 := New()
	c2.Path = make([]string, 0, len(c.Path)+1)
	c2.Path = append(c2.Path, c.Path...)
	c2.Path = append(c2.Path, name)
	c.Children[name] = c2
	c.Start.Then(func(ctx context.Context) error { return c2.Start(ctx) })
	c.Stop.Then(func(ctx context.Context) error { return c2.Stop(ctx) })
	return c2
}
