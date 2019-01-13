// Package mrun extends mctx to include runtime event hooks and tracking of the
// liveness of spawned go-routines.
package mrun

import (
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

type ctxKey int

// Thread spawns a go-routine which executes the given function. When the passed
// in Context is canceled the Context within all threads spawned from it will
// be canceled as well.
//
// See Wait for accompanying functionality.
func Thread(ctx mctx.Context, fn func(mctx.Context) error) {
	futErr := newFutureErr()
	mctx.GetSetMutableValue(ctx, false, ctxKey(0), func(i interface{}) interface{} {
		futErrs, ok := i.([]*futureErr)
		if !ok {
			futErrs = make([]*futureErr, 0, 1)
		}
		return append(futErrs, futErr)
	})

	go func() {
		futErr.set(fn(ctx))
	}()
}

// ErrDone is returned from Wait if cancelCh is closed before all threads have
// returned.
var ErrDone = errors.New("Wait is done waiting")

// Wait blocks until all go-routines spawned using Thread on the passed in
// Context, and all of its children, have returned. Any number of the threads
// may have returned already when Wait is called.
//
// If any of the thread functions returned an error during its runtime Wait will
// return that error. If multiple returned an error only one of those will be
// returned. TODO: Handle multi-errors better.
//
// If cancelCh is not nil and is closed before all threads have returned then
// this function stops waiting and returns ErrDone.
//
// Wait is safe to call in parallel, and will return the same result if called
// multiple times in sequence. If new Thread calls have been made since the last
// Wait call, the results of those calls will be waited upon during subsequent
// Wait calls.
func Wait(ctx mctx.Context, cancelCh <-chan struct{}) error {
	// First wait for all the children, and see if any of them return an error
	children := mctx.Children(ctx)
	for _, childCtx := range children {
		if err := Wait(childCtx, cancelCh); err != nil {
			return err
		}
	}

	futErrs, _ := mctx.MutableValue(ctx, ctxKey(0)).([]*futureErr)
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
