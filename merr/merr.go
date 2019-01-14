// Package merr extends the errors package with features like key-value
// attributes for errors, embedded stacktraces, and multi-errors.
package merr

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var strBuilderPool = sync.Pool{
	New: func() interface{} { return new(strings.Builder) },
}

func putStrBuilder(sb *strings.Builder) {
	sb.Reset()
	strBuilderPool.Put(sb)
}

////////////////////////////////////////////////////////////////////////////////

type val struct {
	visible bool
	val     interface{}
}

func (v val) String() string {
	return fmt.Sprint(v.val)
}

type err struct {
	err  error
	attr map[interface{}]val
}

// attr keys internal to this package
type attrKey string

func wrap(e error, cp bool, skip int) *err {
	er, ok := e.(*err)
	if !ok {
		er := &err{err: e, attr: map[interface{}]val{}}
		if skip >= 0 {
			setStack(er, skip+1)
		}
		return er
	} else if !cp {
		return er
	}

	er2 := &err{
		err:  er.err,
		attr: make(map[interface{}]val, len(er.attr)),
	}
	for k, v := range er.attr {
		er2.attr[k] = v
	}
	if _, ok := er2.attr[attrKeyStack]; !ok && skip >= 0 {
		setStack(er, skip+1)
	}

	return er2
}

// Wrap takes in an error and returns one wrapping it in merr's inner type,
// which embeds information like the stack trace.
func Wrap(e error) error {
	return wrap(e, false, 1)
}

// New returns a new error with the given string as its error string. New
// automatically wraps the error in merr's inner type, which embeds information
// like the stack trace.
func New(str string) error {
	return wrap(errors.New(str), false, 1)
}

// Errorf is like New, but allows for formatting of the string.
func Errorf(str string, args ...interface{}) error {
	return wrap(fmt.Errorf(str, args...), false, 1)
}

func (er *err) visibleAttrs() [][2]string {
	out := make([][2]string, 0, len(er.attr))
	for k, v := range er.attr {
		if !v.visible {
			continue
		}
		out = append(out, [2]string{
			strings.Trim(fmt.Sprintf("%q", k), `"`),
			fmt.Sprint(v.val),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i][0] < out[j][0]
	})

	return out
}

func (er *err) Error() string {
	visAttrs := er.visibleAttrs()
	if len(visAttrs) == 0 {
		return er.err.Error()
	}

	sb := strBuilderPool.Get().(*strings.Builder)
	defer putStrBuilder(sb)

	sb.WriteString(strings.TrimSpace(er.err.Error()))
	for _, attr := range visAttrs {
		k, v := strings.TrimSpace(attr[0]), strings.TrimSpace(attr[1])
		sb.WriteString("\n\t* ")
		sb.WriteString(k)
		sb.WriteString(": ")

		// if there's no newlines then print v inline with k
		if strings.Index(v, "\n") < 0 {
			sb.WriteString(v)
			continue
		}

		for _, vLine := range strings.Split(v, "\n") {
			sb.WriteString("\n\t\t")
			sb.WriteString(strings.TrimSpace(vLine))
		}
	}

	return sb.String()
}

// WithValue returns a copy of the original error, automatically wrapping it if
// the error is not from merr (see Wrap). The returned error has a value set on
// with for the given key.
//
// visible determines whether or not the value is visible in the output of
// Error.
func WithValue(e error, k, v interface{}, visible bool) error {
	er := wrap(e, true, 1)
	er.attr[k] = val{val: v, visible: visible}
	return er
}

// GetValue returns the value embedded in the error for the given key, or nil if
// the error isn't from this package or doesn't have that key embedded.
func GetValue(e error, k interface{}) interface{} {
	return wrap(e, false, -1).attr[k].val
}

// Base takes in an error and checks if it is merr's internal error type. If it
// is then the underlying error which is being wrapped is returned. If it's not
// then the passed in error is returned as-is.
func Base(e error) error {
	if er, ok := e.(*err); ok {
		return er.err
	}
	return e
}

// Equal is a shortcut for Base(e1) == Base(e2).
func Equal(e1, e2 error) bool {
	return Base(e1) == Base(e2)
}
