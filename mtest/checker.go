package mtest

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
)

// CheckerErr represents an test case error which was returned by a Checker Run.
//
// The string form of CheckerErr includes the sequence of Applyers which can be
// copy-pasted directly into Checker's RunCase method's arguments.
type CheckerErr struct {
	// The sequence of applied actions which generated the error
	Applied []Applyer

	// The error returned by the final Action
	Err error
}

func (ce CheckerErr) Error() string {
	typeName := func(a Applyer) string {
		t := fmt.Sprintf("%T", a)
		return strings.SplitN(t, ".", 2)[1] // remove the package name
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "Test case: []mtest.Applyer{\n")
	for _, a := range ce.Applied {
		fmt.Fprintf(buf, "\t%s(%#v),\n", typeName(a), a)
	}
	fmt.Fprintf(buf, "}\n")
	fmt.Fprintf(buf, "Generated error: %s\n", ce.Err)
	return buf.String()
}

// State represents the current state of a Checker run. It can be represented by
// any value convenient and useful to the test.
type State interface{}

// Applyer performs the applies an Action's changes to a State, returning the
// new State. After modifying the State the Applyer should also assert that the
// new State is what it's expected to be, returning an error if it's not.
type Applyer interface {
	Apply(State) (State, error)
}

// Action describes a change which can take place on a state. It must contain an
// Applyer which will peform the actual change.
type Action struct {
	Applyer

	// Weight is used when the Checker is choosing which Action to apply on
	// every loop. If not set it is assumed to be 1, and can be increased
	// further to increase its chances of being picked.
	Weight uint64

	// Incomplete can be set to true to indicate that this Action should never
	// be the last Action applied, even if that means the maxDepth of the Run is
	// gone over.
	Incomplete bool

	// Terminate can be set to true to indicate that this Action should always
	// be the last Action applied, even if that means the maxDepth hasn't been
	// reached yet.
	Terminate bool
}

// Checker implements a very basic property checker. It generates random test
// cases, attempting to find and print out failing ones.
type Checker struct {
	// Init returns the initial state of the test. It should always return the
	// exact same value.
	Init func() State

	// Actions returns possible Actions which could be applied to the given
	// State. This is called after Init and after every subsequent Action is
	// applied.
	Actions func(State) []Action
}

// Run performs RunOnce in a loop until maxDuration has elapsed.
func (c Checker) Run(maxDepth int, maxDuration time.Duration) error {
	doneTimer := time.After(maxDuration)
	for {
		select {
		case <-doneTimer:
			return nil
		default:
		}

		if err := c.RunOnce(maxDepth); err != nil {
			return err
		}
	}
}

// RunOnce generates a single sequence of Actions and applies them in order,
// returning nil once the number of Actions performed has reached maxDepth or a
// CheckErr if an error is returned.
func (c Checker) RunOnce(maxDepth int) error {
	s := c.Init()
	applied := make([]Applyer, 0, maxDepth)
	for {
		actions := c.Actions(s)
		action := mrand.Element(actions, func(i int) uint64 {
			if actions[i].Weight == 0 {
				return 1
			}
			return actions[i].Weight
		}).(Action)

		var err error
		s, err = action.Apply(s)
		applied = append(applied, action.Applyer)

		if err != nil {
			return CheckerErr{
				Applied: applied,
				Err:     err,
			}
		} else if action.Incomplete {
			continue
		} else if action.Terminate || len(applied) >= maxDepth {
			return nil
		}
	}
}

// RunCase performs a single sequence of Applyers in order, returning a CheckErr
// if one is returned by one of the Applyers.
func (c Checker) RunCase(aa ...Applyer) error {
	s := c.Init()
	for i := range aa {
		var err error
		if s, err = aa[i].Apply(s); err != nil {
			return CheckerErr{
				Applied: aa[:i+1],
				Err:     err,
			}
		}
	}
	return nil
}
