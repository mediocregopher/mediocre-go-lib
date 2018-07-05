// Package massert implements an assertion framework which is useful in tests.
package massert

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"text/tabwriter"
)

// AssertErr is an error returned by Assertions which have failed, containing
// information about both the reason for failure and the Assertion itself.
type AssertErr struct {
	Err       error     // The error which occurred
	Assertion Assertion // The Assertion which failed
}

func fmtBlock(str string) string {
	if strings.Index(str, "\n") == -1 {
		return str
	}
	return "\n\t" + strings.Replace(str, "\n", "\n\t", -1) + "\n"
}

func fmtStack(frames []runtime.Frame) string {
	buf := new(bytes.Buffer)
	tw := tabwriter.NewWriter(buf, 0, 4, 2, ' ', 0)
	for _, frame := range frames {
		file := filepath.Base(frame.File)
		fmt.Fprintf(tw, "%s:%d\t%s\n", file, frame.Line, frame.Function)
	}
	if err := tw.Flush(); err != nil {
		panic(err) // fuck it
	}
	return buf.String()
}

func (ae AssertErr) Error() string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "Assertion: %s\n", fmtBlock(ae.Assertion.Description()))
	fmt.Fprintf(buf, "Error: %s\n", fmtBlock(ae.Err.Error()))
	fmt.Fprintf(buf, "Stack: %s\n", fmtBlock(fmtStack(ae.Assertion.Stack())))
	return buf.String()
}

////////////////////////////////////////////////////////////////////////////////

// Assertion is an entity which will make some kind of assertion and produce an
// error if that assertion does not hold true. The error returned will generally
// be of type AssertErr.
type Assertion interface {
	Assert() error
	Description() string // A description of the Assertion

	// Returns the callstack of where the Assertion was created, ordered from
	// closest to farthest. This may not necessarily contain the entire
	// callstack if that would be inconveniently cumbersome.
	Stack() []runtime.Frame
}

const maxStackLen = 8

type assertion struct {
	fn    func() error
	descr string
	stack []runtime.Frame
}

func newAssertion(assertFn func() error, descr string, skip int) Assertion {
	pcs := make([]uintptr, maxStackLen)
	// first skip is for runtime.Callers, second is for newAssertion, third is
	// for whatever is calling newAssertion
	numPCs := runtime.Callers(skip+3, pcs)
	stack := make([]runtime.Frame, 0, maxStackLen)
	frames := runtime.CallersFrames(pcs[:numPCs])
	for {
		frame, more := frames.Next()
		stack = append(stack, frame)
		if !more || len(stack) == maxStackLen {
			break
		}
	}

	a := &assertion{
		descr: descr,
		stack: stack,
	}
	a.fn = func() error {
		err := assertFn()
		if err == nil {
			return nil
		}
		return AssertErr{
			Err:       err,
			Assertion: a,
		}
	}
	return a
}

func (a *assertion) Assert() error {
	return a.fn()
}

func (a *assertion) Description() string {
	return a.descr
}

func (a *assertion) Stack() []runtime.Frame {
	return a.stack
}

// Assertions represents a set of Assertions which can be tested all at once.
type Assertions []Assertion

// New returns an empty set of Assertions which can be Add'd to.
func New() Assertions {
	return make(Assertions, 0, 8)
}

// Add adds the given Assertion to the set.
func (aa *Assertions) Add(a Assertion) {
	(*aa) = append(*aa, a)
}

// Assert performs the Assert method of each of the set's Assertions
// sequentially, stopping at the first error and generating a new one which
// includes the Assertion's string and stack information.
func (aa Assertions) Assert() error {
	for _, a := range aa {
		if err := a.Assert(); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Assertion wrappers

// if the Assertion is a wrapper for another, this makes sure that if the
// underlying one returns an AssertErr that this Assertion is what ends up in
// that AssertErr
type wrap struct {
	Assertion
}

func (wa wrap) Assert() error {
	err := wa.Assertion.Assert()
	if err == nil {
		return nil
	}
	ae := err.(AssertErr)
	ae.Assertion = wa.Assertion
	return ae
}

type descrWrap struct {
	Assertion
	descr string
}

func (dw descrWrap) Description() string {
	return dw.descr
}

// Comment prepends a formatted string to the given Assertions string
// description.
func Comment(a Assertion, msg string, args ...interface{}) Assertion {
	msg = strings.TrimSpace(msg)
	descr := fmt.Sprintf("/* "+msg+" */\n", args...)
	descr += a.Description()
	return wrap{descrWrap{Assertion: a, descr: descr}}
}

type not struct {
	Assertion
}

func (n not) Assert() error {
	if err := n.Assertion.Assert(); err == nil {
		return AssertErr{
			Err:       errors.New("assertion should have failed"),
			Assertion: n,
		}
	}
	return nil
}

func (n not) Description() string {
	return "not(" + fmtBlock(n.Assertion.Description()) + ")"
}

// Not negates an Assertion, so that it fails if the given Assertion does not,
// and vice-versa.
func Not(a Assertion) Assertion {
	return not{Assertion: a}
}

////////////////////////////////////////////////////////////////////////////////

var typeOfInt64 = reflect.TypeOf(int64(0))

func toStr(i interface{}) string {
	return fmt.Sprintf("%T(%#v)", i, i)
}

// Equal asserts that the two values given are equal. The equality checking
// done is to some degree fuzzy in the following ways:
//
// * All pointers are dereferenced.
// * All ints and uints are converted to int64.
//
func Equal(a, b interface{}) Assertion {
	normalize := func(v reflect.Value) reflect.Value {
		v = reflect.Indirect(v)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v = v.Convert(typeOfInt64)
		}
		return v
	}

	fn := func() error {
		aV, bV := reflect.ValueOf(a), reflect.ValueOf(b)
		aV, bV = normalize(aV), normalize(bV)
		if !reflect.DeepEqual(aV.Interface(), bV.Interface()) {
			return errors.New("not equal")
		}
		return nil
	}

	return newAssertion(fn, toStr(a)+" == "+toStr(b), 0)
}

// Exactly asserts that the two values are exactly equal, and uses the
// reflect.DeepEquals function to determine if they are.
func Exactly(a, b interface{}) Assertion {
	return newAssertion(func() error {
		if !reflect.DeepEqual(a, b) {
			return errors.New("not exactly equal")
		}
		return nil
	}, toStr(a)+" === "+toStr(b), 0)
}
