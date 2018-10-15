// Package massert implements an assertion framework which is useful in tests.
package massert

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
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

func fmtMultiBlock(prefix string, elems ...string) string {
	if len(elems) == 0 {
		return prefix + "()"
	} else if len(elems) == 1 {
		return prefix + "(" + fmtBlock(elems[0]) + ")"
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s(\n", prefix)
	for _, el := range elems {
		elStr := "\t" + strings.Replace(el, "\n", "\n\t", -1)
		fmt.Fprintf(buf, "%s,\n", elStr)
	}
	fmt.Fprintf(buf, ")")
	return buf.String()
}

func fmtMultiDescr(prefix string, aa ...Assertion) string {
	descrs := make([]string, len(aa))
	for i := range aa {
		descrs[i] = aa[i].Description()
	}
	return fmtMultiBlock(prefix, descrs...)
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
	fmt.Fprintf(buf, "\n")
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
		} else if ae, ok := err.(AssertErr); ok {
			return ae
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

////////////////////////////////////////////////////////////////////////////////

// Fatal is a convenience function which performs the Assertion and calls Fatal
// on the testing.T instance if the assertion fails.
func Fatal(t *testing.T, a Assertion) {
	if err := a.Assert(); err != nil {
		t.Fatal(err)
	}
}

// Error is a convenience function which performs the Assertion and calls Error
// on the testing.T instance if the assertion fails.
func Error(t *testing.T, a Assertion) {
	if err := a.Assert(); err != nil {
		t.Error(err)
	}
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

// Comment prepends a formatted string to the given Assertion's string
// description.
func Comment(a Assertion, msg string, args ...interface{}) Assertion {
	msg = strings.TrimSpace(msg)
	descr := fmt.Sprintf("/* "+msg+" */\n", args...)
	descr += a.Description()
	return wrap{descrWrap{Assertion: a, descr: descr}}
}

// Not negates an Assertion, so that it fails if the given Assertion does not,
// and vice-versa.
func Not(a Assertion) Assertion {
	fn := func() error {
		if err := a.Assert(); err == nil {
			return errors.New("assertion should have failed")
		}
		return nil
	}
	return newAssertion(fn, fmtMultiDescr("Not", a), 0)
}

// Any asserts that at least one of the given Assertions succeeds.
func Any(aa ...Assertion) Assertion {
	fn := func() error {
		for _, a := range aa {
			if err := a.Assert(); err == nil {
				return nil
			}
		}
		return errors.New("no assertions succeeded")
	}
	return newAssertion(fn, fmtMultiDescr("Any", aa...), 0)
}

// AnyOne asserts that exactly one of the given Assertions succeeds.
func AnyOne(aa ...Assertion) Assertion {
	fn := func() error {
		any := -1
		for i, a := range aa {
			if err := a.Assert(); err == nil {
				if any >= 0 {
					return fmt.Errorf("assertions indices %d and %d both succeeded", any, i)
				}
				any = i
			}
		}
		if any == -1 {
			return errors.New("no assertions succeeded")
		}
		return nil
	}
	return newAssertion(fn, fmtMultiDescr("AnyOne", aa...), 0)
}

// All asserts that at all of the given Assertions succeed. Its Assert method
// will return the error of whichever Assertion failed.
func All(aa ...Assertion) Assertion {
	fn := func() error {
		for _, a := range aa {
			if err := a.Assert(); err != nil {
				// newAssertion will pass this error through, so that its
				// description and callstack is what gets displayed as the
				// error. This isn't totally consistent with Any's behavior, but
				// it's fine.
				return err
			}
		}
		return nil
	}
	return newAssertion(fn, fmtMultiDescr("All", aa...), 0)
}

// None asserts that all of the given Assertions fail.
//
// NOTE this is functionally equivalent to doing `Not(Any(aa...))`, but the
// error returned is more helpful.
func None(aa ...Assertion) Assertion {
	fn := func() error {
		for _, a := range aa {
			if err := a.Assert(); err == nil {
				return AssertErr{
					Err:       errors.New("assertion should not have succeeded"),
					Assertion: a,
				}
			}
		}
		return nil
	}
	return newAssertion(fn, fmtMultiDescr("None", aa...), 0)
}

////////////////////////////////////////////////////////////////////////////////

func toStr(i interface{}) string {
	return fmt.Sprintf("%T(%#v)", i, i)
}

// Equal asserts that the two values are exactly equal, and uses the
// reflect.DeepEqual function to determine if they are.
//
// TODO this does not currently handle the case of creating the Assertion using
// a reference type (like a map), changing one of the map's keys, and then
// calling Assert.
func Equal(a, b interface{}) Assertion {
	return newAssertion(func() error {
		if !reflect.DeepEqual(a, b) {
			return errors.New("not exactly equal")
		}
		return nil
	}, toStr(a)+" == "+toStr(b), 0)
}

// Nil asserts that the value is nil. This assertion works both if the value is
// the untyped nil value (e.g. `Nil(nil)`) or if it's a typed nil value (e.g.
// `Nil([]byte(nil))`).
func Nil(i interface{}) Assertion {
	return newAssertion(func() error {
		if i == nil {
			return nil
		}
		v := reflect.ValueOf(i)
		switch v.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface,
			reflect.Map, reflect.Ptr, reflect.Slice:
			if v.IsNil() {
				return nil
			}
		default:
		}
		return errors.New("not nil")
	}, toStr(i)+" is nil", 0)
}

type setKV struct {
	k, v interface{}
}

func toSet(i interface{}, keyedMap bool) ([]interface{}, error) {
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		vv := make([]interface{}, v.Len())
		for i := range vv {
			vv[i] = v.Index(i).Interface()
		}
		return vv, nil
	case reflect.Map:
		keys := v.MapKeys()
		vv := make([]interface{}, len(keys))
		for i := range keys {
			if keyedMap {
				vv[i] = setKV{
					k: keys[i].Interface(),
					v: v.MapIndex(keys[i]).Interface(),
				}
			} else {
				vv[i] = v.MapIndex(keys[i]).Interface()
			}
		}
		return vv, nil
	default:
		return nil, fmt.Errorf("cannot turn value of type %s into a set", v.Type())
	}
}

// Subset asserts that the given subset is a subset of the given set. Both must
// be of the same type and may be arrays, slices, or maps.
func Subset(set, subset interface{}) Assertion {
	if reflect.TypeOf(set) != reflect.TypeOf(subset) {
		panic(errors.New("set and subset aren't of same type"))
	}

	setVV, err := toSet(set, true)
	if err != nil {
		panic(err)
	}
	subsetVV, err := toSet(subset, true)
	if err != nil {
		panic(err)
	}
	return newAssertion(func() error {
		// this is obviously not the most efficient way to do this
	outer:
		for i := range subsetVV {
			for j := range setVV {
				if reflect.DeepEqual(setVV[j], subsetVV[i]) {
					continue outer
				}
			}
			return fmt.Errorf("missing element %s", toStr(subsetVV[i]))
		}
		return nil
	}, toStr(set)+" has subset "+toStr(subset), 0)
}

// Has asserts that the given set has the given element as a value in it. The
// set may be an array, a slice, or a map, and if it's a map then the elem will
// need to be a value in it.
func Has(set, elem interface{}) Assertion {
	setVV, err := toSet(set, false)
	if err != nil {
		panic(err)
	}

	return newAssertion(func() error {
		for i := range setVV {
			if reflect.DeepEqual(setVV[i], elem) {
				return nil
			}
		}
		return errors.New("value not in set")
	}, toStr(set)+" has value "+toStr(elem), 0)
}

// HasKey asserts that the given set (which must be a map type) has the given
// element as a key in it.
func HasKey(set, elem interface{}) Assertion {
	if v := reflect.ValueOf(set); v.Kind() != reflect.Map {
		panic(fmt.Errorf("type %s is not a map", v.Type()))
	}
	setVV, err := toSet(set, true)
	if err != nil {
		panic(err)
	}
	return newAssertion(func() error {
		for _, kv := range setVV {
			if reflect.DeepEqual(kv.(setKV).k, elem) {
				return nil
			}
		}
		return errors.New("value not a key in the map")
	}, toStr(set)+" has key "+toStr(elem), 0)
}

// Len asserts that the given set has the given number of elements in it. The
// set may be an array, a slice, or a map. A nil value'd set is considered to be
// a length of zero.
func Len(set interface{}, length int) Assertion {
	setVV, err := toSet(set, false)
	if err != nil {
		panic(err)
	}

	return newAssertion(func() error {
		if len(setVV) != length {
			return fmt.Errorf("set not correct length, is %d", len(setVV))
		}
		return nil
	}, toStr(set)+" has length "+strconv.Itoa(length), 0)
}

// TODO ChanRead(ch interface{}, within time.Duration, callback func(interface{}) error)
// TODO ChanBlock(ch interface{}, for time.Duration)
// TODO ChanClosed(ch interface{})
