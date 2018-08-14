// Package mchk implements a framework for writing property checker tests, where
// test cases are generated randomly and performed, and failing test cases are
// output in a way so as to easily be able to rerun them.
//
// The central type of the package is Checker. For every Run call on Checker a
// new initial State is generated, and then an Action is generated off of that.
// The Action is applied to the State to obtain a new State, and a new Action is
// generated from there, and so on. If any Action fails it is output along with
// all of the Actions leading up to it.
package mchk

import (
	"bytes"
	"fmt"
	"time"
)

// RunErr represents an test case error which was returned by a Checker Run.
//
// The string form of RunErr includes the sequence of Params which can be
// copy-pasted directly into Checker's RunCase method's arguments.
type RunErr struct {
	// The sequence of Action Params which generated the error
	Params []Params

	// The error returned by the final Action
	Err error
}

func (ce RunErr) Error() string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "Test case: []mtest.Params{\n")
	for _, p := range ce.Params {
		fmt.Fprintf(buf, "\t%#v,\n", p)
	}
	fmt.Fprintf(buf, "}\n")
	fmt.Fprintf(buf, "Generated error: %s\n", ce.Err)
	return buf.String()
}

// State represents the current state of a Checker run. It can be any value
// convenient and useful to the test.
type State interface{}

// Params represent the parameters to an Action used during a Checker run. It
// should be a static value, meaning no pointers or channels.
type Params interface{}

// Action describes a change which can take place on a state.
type Action struct {
	// Params are defined by the test and affect the behavior of the Action.
	Params Params

	// Incomplete can be set to true to indicate that this Action should never
	// be the last Action applied, even if that means the length of the Run goes
	// over MaxLength.
	Incomplete bool

	// Terminate can be set to true to indicate that this Action should always
	// be the last Action applied, even if the Run's length hasn't reached
	// MaxLength yet.
	Terminate bool
}

// Checker implements a very basic property checker. It generates random test
// cases, attempting to find and print out failing ones.
type Checker struct {
	// Init returns the initial state of the test. It should always return the
	// exact same value.
	Init func() State

	// Next returns a new Action which can be Apply'd to the given State. This
	// function should not modify the State in any way.
	Next func(State) Action

	// Apply performs the Action's changes to a State, returning the new State.
	// After modifying the State this function should also assert that the new
	// State is what it's expected to be, returning an error if it's not.
	Apply func(State, Action) (State, error)

	// Cleanup is an optional function which can perform any necessary cleanup
	// operations on the State. This is called even on error.
	Cleanup func(State)

	// MaxLength indicates the maximum number of Actions which can be strung
	// together in a single Run. Defaults to 10 if not set.
	MaxLength int
}

func (c Checker) withDefaults() Checker {
	if c.MaxLength == 0 {
		c.MaxLength = 10
	}
	return c
}

// RunFor performs Runs in a loop until maxDuration has elapsed.
func (c Checker) RunFor(maxDuration time.Duration) error {
	doneTimer := time.After(maxDuration)
	for {
		select {
		case <-doneTimer:
			return nil
		default:
		}

		if err := c.Run(); err != nil {
			return err
		}
	}
}

// Run generates a single sequence of Actions and applies them in order,
// returning nil once the number of Actions performed has reached MaxLength or a
// CheckErr if an error is returned.
func (c Checker) Run() error {
	c = c.withDefaults()
	s := c.Init()
	params := make([]Params, 0, c.MaxLength)
	for {
		action := c.Next(s)
		var err error
		s, err = c.Apply(s, action)
		params = append(params, action.Params)

		if err != nil {
			return RunErr{
				Params: params,
				Err:    err,
			}
		} else if action.Incomplete {
			continue
		} else if action.Terminate || len(params) >= c.MaxLength {
			return nil
		}
	}
}

// RunCase performs a single sequence of Actions with the given Params.
func (c Checker) RunCase(params ...Params) error {
	s := c.Init()
	if c.Cleanup != nil {
		// wrap in a function so we don't capture the value of s right here
		defer func() {
			c.Cleanup(s)
		}()
	}
	for i := range params {
		var err error
		if s, err = c.Apply(s, Action{Params: params[i]}); err != nil {
			return RunErr{
				Params: params[:i+1],
				Err:    err,
			}
		}
	}
	return nil
}
