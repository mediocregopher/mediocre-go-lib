// Package merr extends the errors package with features like key-value
// attributes for errors, embedded stacktraces, and multi-errors.
//
// merr functions takes in generic errors of the built-in type. The returned
// errors are wrapped by a type internal to merr, and appear to also be of the
// generic error type.
//
// As is generally recommended for go projects, errors.Is and errors.As should
// be used for equality checking.
package merr

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/mediocregopher/mediocre-go-lib/v2/mctx"
)

var strBuilderPool = sync.Pool{
	New: func() interface{} { return new(strings.Builder) },
}

func putStrBuilder(sb *strings.Builder) {
	sb.Reset()
	strBuilderPool.Put(sb)
}

////////////////////////////////////////////////////////////////////////////////

type annotateKey string

// Error wraps an error such that contextual and stacktrace information is
// captured alongside that error.
type Error struct {
	Err        error
	Ctx        context.Context
	Stacktrace Stacktrace
}

// Error implements the method for the error interface.
func (e Error) Error() string {
	sb := strBuilderPool.Get().(*strings.Builder)
	defer putStrBuilder(sb)
	sb.WriteString(strings.TrimSpace(e.Err.Error()))

	annotations := make(mctx.Annotations)
	mctx.EvaluateAnnotations(e.Ctx, annotations)

	annotations[annotateKey("line")] = e.Stacktrace.String()

	for _, kve := range annotations.StringSlice(true) {
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

// Unwrap implements the method for the errors package.
func (e Error) Unwrap() error {
	return e.Err
}

// WrapSkip is like Wrap but also allows for skipping extra stack frames when
// embedding the stack into the error.
func WrapSkip(ctx context.Context, err error, skip int) error {
	if err == nil {
		return nil
	}

	if e := (Error{}); errors.As(err, &e) {
		e.Err = err
		e.Ctx = mctx.MergeAnnotations(e.Ctx, ctx)
		return e
	}

	return Error{
		Err:        err,
		Ctx:        ctx,
		Stacktrace: newStacktrace(skip + 1),
	}
}

// Wrap returns a copy of the given error wrapped in an Error. If the given
// error is already wrapped in an *Error then the given context is merged into
// that one with mctx.MergeAnnotations instead.
//
// Wrapping nil returns nil.
func Wrap(ctx context.Context, err error) error {
	return WrapSkip(ctx, err, 1)
}

// New is a shortcut for:
//	merr.WrapSkip(ctx, errors.New(str), 1)
func New(ctx context.Context, str string) error {
	return WrapSkip(ctx, errors.New(str), 1)
}
