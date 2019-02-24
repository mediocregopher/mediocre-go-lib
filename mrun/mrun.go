// Package mrun extends mctx to include runtime event hooks and tracking of the
// liveness of spawned go-routines.
package mrun

import (
	"context"
	"errors"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
)

type futureErr struct {
	doneCh chan struct{}
	err    error
}

func newFutureErr() *futureErr {
	return &futureErr{
		doneCh: make(chan struct{}),
	}
}

func (fe *futureErr) get(cancelCh <-chan struct{}) (error, bool) {
	select {
	case <-fe.doneCh:
		return fe.err, true
	case <-cancelCh:
		return nil, false
	}
}

func (fe *futureErr) set(err error) {
	fe.err = err
	close(fe.doneCh)
}

type threadCtxKey int

// WithThreads spawns n go-routines, each of which executes the given function. The
// returned Context tracks these go-routines, and can then be passed into the
// Wait function to block until the spawned go-routines all return.
func WithThreads(ctx context.Context, n uint, fn func() error) context.Context {
	// I dunno why this would happen, but it wouldn't actually hurt anything
	if n == 0 {
		return ctx
	}

	oldFutErrs, _ := ctx.Value(threadCtxKey(0)).([]*futureErr)
	futErrs := make([]*futureErr, len(oldFutErrs), len(oldFutErrs)+int(n))
	copy(futErrs, oldFutErrs)

	for i := uint(0); i < n; i++ {
		futErr := newFutureErr()
		futErrs = append(futErrs, futErr)

		go func() {
			futErr.set(fn())
		}()
	}

	return context.WithValue(ctx, threadCtxKey(0), futErrs)
}

// ErrDone is returned from Wait if cancelCh is closed before all threads have
// returned.
var ErrDone = errors.New("Wait is done waiting")

// Wait blocks until all go-routines spawned using WithThreads on the passed in
// Context (and its predecessors) have returned. Any number of the go-routines
// may have returned already when Wait is called, and not all go-routines need
// be from the same WithThreads call.
//
// If any of the thread functions returned an error during its runtime Wait will
// return that error. If multiple returned an error only one of those will be
// returned. TODO: Handle multi-errors better.
//
// If cancelCh is not nil and is closed before all threads have returned then
// this function stops waiting and returns ErrDone.
//
// Wait is safe to call in parallel, and will return the same result if called
// multiple times.
func Wait(ctx context.Context, cancelCh <-chan struct{}) error {
	// First wait for all the children, and see if any of them return an error
	children := mctx.Children(ctx)
	for _, childCtx := range children {
		if err := Wait(childCtx, cancelCh); err != nil {
			return err
		}
	}

	futErrs, _ := ctx.Value(threadCtxKey(0)).([]*futureErr)
	for _, futErr := range futErrs {
		err, ok := futErr.get(cancelCh)
		if !ok {
			return ErrDone
		} else if err != nil {
			return err
		}
	}

	return nil
}
