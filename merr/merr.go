// Package merr extends the errors package with features like key-value
// attributes for errors, embedded stacktraces, and multi-errors.
//
// merr functions takes in generic errors of the built-in type. The returned
// errors are wrapped by a type internal to merr, and appear to also be of the
// generic error type. This means that equality checking will not work, unless
// the Base function is used. If any functions are given nil they will also
// return nil.
package merr

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
)

var strBuilderPool = sync.Pool{
	New: func() interface{} { return new(strings.Builder) },
}

func putStrBuilder(sb *strings.Builder) {
	sb.Reset()
	strBuilderPool.Put(sb)
}

////////////////////////////////////////////////////////////////////////////////

type err struct {
	err  error
	attr map[interface{}]interface{}
}

// attr keys internal to this package
type attrKey int

const (
	attrKeyCtx attrKey = iota
)

func wrap(e error, cp bool) *err {
	if e == nil {
		return nil
	}

	er, ok := e.(*err)
	if !ok {
		return &err{
			err:  e,
			attr: make(map[interface{}]interface{}, 1),
		}
	} else if !cp {
		return er
	}

	er2 := &err{
		err:  er.err,
		attr: make(map[interface{}]interface{}, len(er.attr)+1),
	}
	for k, v := range er.attr {
		er2.attr[k] = v
	}

	return er2
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

// WithValue returns a copy of the original error, automatically wrapping it if
// the error is not from merr (see Wrap). The returned error has an attribute
// value set on it for the given key.
func WithValue(e error, k, v interface{}) error {
	if e == nil {
		return nil
	}
	er := wrap(e, true)
	if er.attr == nil {
		er.attr = map[interface{}]interface{}{}
	}
	er.attr[k] = v
	return er
}

// Value returns the value embedded in the error for the given key, or nil if
// the error isn't from this package or doesn't have that key embedded.
func Value(e error, k interface{}) interface{} {
	if e == nil {
		return nil
	}
	return wrap(e, false).attr[k]
}

// WrapSkip is like Wrap but also allows for skipping extra stack frames when
// embedding the stack into the error.
func WrapSkip(e error, skip int, ctxs ...context.Context) error {
	if e == nil {
		return nil
	}

	er := wrap(e, true)
	if _, ok := getStack(er); !ok {
		setStack(er, skip+1)
	}

	ctx, _ := er.attr[attrKeyCtx].(context.Context)
	if ctx != nil {
		ctx = mctx.MergeAnnotationsInto(ctx, ctxs...)
	} else if len(ctxs) > 0 {
		ctx = mctx.MergeAnnotations(ctxs...)
	}

	er.attr[attrKeyCtx] = ctx
	return er
}

// Wrap returns a copy of the given error wrapped in merr's inner type. It will
// perform an mctx.MergeAnnotations on the given Contexts to create a new
// Context, and embed that in the returned error. If the given error already has
// an embedded Context then ctxs will be merged into that.
//
// This function automatically embeds stack information into the error as it's
// being stored, using WithStack, unless the error already has stack information
// in it.
//
// Wrapping nil returns nil.
func Wrap(e error, ctx ...context.Context) error {
	return WrapSkip(e, 1, ctx...)
}

// New is a shortcut for:
//	merr.Wrap(errors.New(str), ctxs...)
func New(str string, ctxs ...context.Context) error {
	return WrapSkip(errors.New(str), 1, ctxs...)
}

type annotateKey string

func ctx(e error) context.Context {
	ctx, _ := Value(e, attrKeyCtx).(context.Context)
	if ctx == nil {
		ctx = context.Background()
	}

	if stack, ok := Stack(e); ok {
		ctx = mctx.Annotate(ctx, annotateKey("errLoc"), stack.String())
	}
	return ctx
}

// Context returns the Context embedded in this error from the last call to Wrap
// or New. If none is embedded this uses context.Background().
//
// The returned Context will have annotated on it (see mctx.Annotate) the
// underlying error's string (as returned by Error()) and the error's stack
// location. Stack locations are automatically added by New and Wrap via
// WithStack.
//
// If this error is nil this returns context.Background().
func Context(e error) context.Context {
	if e == nil {
		return context.Background()
	}
	ctx := ctx(e)
	ctx = mctx.Annotate(ctx, annotateKey("err"), Base(e).Error())
	return ctx
}

func (er *err) Error() string {
	ctx := ctx(er)

	sb := strBuilderPool.Get().(*strings.Builder)
	defer putStrBuilder(sb)
	sb.WriteString(strings.TrimSpace(er.err.Error()))

	annotations := mctx.Annotations(ctx).StringSlice(true)
	for _, kve := range annotations {
		k, v := strings.TrimSpace(kve[0]), strings.TrimSpace(kve[1])
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

// Equal is a shortcut for Base(e1) == Base(e2).
func Equal(e1, e2 error) bool {
	return Base(e1) == Base(e2)
}
