// Package mcfg provides a simple foundation for complex service/binary
// configuration, initialization, and destruction
package mcfg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
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
		err := h2(ctx)
		if err := <-errCh; err != nil {
			return err
		}
		return err
	}
}

// ParamValue describes a value for a parameter which has been parsed by a
// Source
type ParamValue struct {
	Param
	Path  []string // nil if root
	Value json.RawMessage
}

// Source parses ParamValues out of a particular configuration source
type Source interface {
	Parse(*Cfg) ([]ParamValue, error)
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

	// Default 2 minutes. Timeout within which the Start Hook (and the Start
	// Hooks of all children of this Cfg) must complete.
	StartTimeout time.Duration

	// Stop hook is performed on interrupt signal, and should stop all
	// go-routines and close all resource handlers created during Start
	Stop Hook

	// Default 30 seconds. Timeout within which the Stop Hook (and the Stop
	// Hooks of all children of this Cfg) must complete.
	StopTimeout time.Duration
}

// New initializes and returns an empty Cfg with default values filled in
func New() *Cfg {
	return &Cfg{
		Children:     map[string]*Cfg{},
		Params:       map[string]Param{},
		Start:        Nop(),
		StartTimeout: 2 * time.Minute,
		Stop:         Nop(),
		StopTimeout:  30 * time.Second,
	}
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

	// first dedupe the params. We use this param struct as the key by which to
	// dedupe by. Its use depends on the json.Marshaler always ordering fields
	// in a marshaled struct the same way, which isn't the best assumption but
	// it's ok for now
	type param struct {
		Path []string `json:",omitempty"`
		Name string
	}

	pvM := map[string]ParamValue{}
	for _, pv := range pvs {
		keyB, err := json.Marshal(param{Path: pv.Path, Name: pv.Name})
		if err != nil {
			return err
		}
		pvM[string(keyB)] = pv
	}

	// check for required params, again using the param struct and the existing
	// pvM
	var requiredParams func(*Cfg) []param
	requiredParams = func(c *Cfg) []param {
		var out []param
		for _, p := range c.Params {
			if !p.Required {
				continue
			}
			out = append(out, param{Path: c.Path, Name: p.Name})
		}
		for _, child := range c.Children {
			out = append(out, requiredParams(child)...)
		}
		return out
	}

	for _, reqP := range requiredParams(c) {
		keyB, err := json.Marshal(reqP)
		if err != nil {
			return err
		} else if _, ok := pvM[string(keyB)]; !ok {
			return fmt.Errorf("param %s is required but wasn't populated by any configuration source", keyB)
		}
	}

	for _, pv := range pvM {
		if pv.Into == nil {
			continue
		}
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

	startCtx, cancel := context.WithTimeout(ctx, c.StartTimeout)
	defer cancel()
	return c.Start(startCtx)
}

// Run blocks while performing all steps of a Cfg run. The steps, in order, are;
// * Populate all configuration parameters
// * Recursively perform Start hooks, depth first
// * Block till the passed in context is cancelled
// * Recursively perform Stop hooks, depth first
//
// If any step returns an error then everything returns that error immediately.
//
// A caveat about Run is that the error case doesn't leave a lot of room for a
// proper cleanup. If you care about that sort of thing you'll need to handle it
// yourself.
func (c *Cfg) Run(ctx context.Context, src Source) error {
	if err := c.runPreBlock(ctx, src); err != nil {
		return err
	}

	<-ctx.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), c.StopTimeout)
	defer cancel()
	return c.Stop(stopCtx)
}

// TestRun is like Run, except it's intended to only be used during tests to
// initialize other entities which are going to actually be tested. It assumes
// all default configuration param values, and will return after the Start hook
// has completed. It will panic on any errors.
func (c *Cfg) TestRun() {
	if err := c.runPreBlock(context.Background(), nil); err != nil {
		panic(err)
	}
}

// Child returns a sub-Cfg of the callee with the given name. The name will be
// prepended to all configuration options created in the returned sub-Cfg, and
// must not be empty.
func (c *Cfg) Child(name string) *Cfg {
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
