// Package mrun TODO
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
// in Context is cancceled the Context within all threads spawned from it will
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

type ctxEventKeyWrap struct {
	key interface{}
}

// Hook describes a function which can be registered to trigger on an event via
// the OnEvent function.
type Hook func(mctx.Context) error

// OnEvent registers a Hook under a typed key. The Hook will be called when
// TriggerEvent is called with that same key. Multiple Hooks can be registered
// for the same key, and will be called sequentially when triggered.
//
// OnEvent registers Hooks onto the root of the given Context. Therefore, Hooks
// will be triggered in the global order they were registered (i.e. if a Hook is
// registered on a Context, then one registered on a child of that Context, then
// another on the original Context again, the three Hooks will be triggered in
// the order: parent, child, parent).
//
// Hooks will be called with whatever Context is passed into TriggerEvent.
func OnEvent(ctx mctx.Context, key interface{}, hook Hook) {
	ctx = mctx.Root(ctx)
	mctx.GetSetMutableValue(ctx, false, ctxEventKeyWrap{key}, func(v interface{}) interface{} {
		hooks, _ := v.([]Hook)
		return append(hooks, hook)
	})
}

// TriggerEvent causes all Hooks registered with OnEvent under the given key to
// be called sequentially, using the given Context as their input. The given
// Context does not need to be the root Context (see OnEvent).
//
// If any Hook returns an error no further Hooks will be called and that error
// will be returned.
//
// TriggerEvent causes all Hooks which were called to be de-registered. If an
// error caused execution to stop prematurely then any Hooks which were not
// called will remain registered.
func TriggerEvent(ctx mctx.Context, key interface{}) error {
	rootCtx := mctx.Root(ctx)
	var err error
	mctx.GetSetMutableValue(rootCtx, false, ctxEventKeyWrap{key}, func(i interface{}) interface{} {
		hooks, _ := i.([]Hook)
		for _, hook := range hooks {
			hooks = hooks[1:]

			// err here is the var outside GetSetMutableValue, we lift it out
			if err = hook(ctx); err != nil {
				break
			}
		}

		// if there was an error then we want to keep all the hooks which
		// weren't called. If there wasn't we want to reset the value to nil so
		// the slice doesn't grow unbounded.
		if err != nil {
			return hooks
		}
		return nil
	})
	return err
}

type builtinEvent int

const (
	start builtinEvent = iota
	stop
)

// OnStart registers the given Hook to run when Start is called. This is a
// special case of OnEvent.
//
// As a convention Hooks running on the start event should block only as long as
// it takes to ensure that whatever is running can do so successfully. For
// short-lived tasks this isn't a problem, but long-lived tasks (e.g. a web
// server) will want to use the Hook only to initialize, and spawn off a
// go-routine to do their actual work. Long-lived tasks should set themselves up
// to stop on the stop event (see OnStop).
func OnStart(ctx mctx.Context, hook Hook) {
	OnEvent(ctx, start, hook)
}

// Start runs all Hooks registered using OnStart. This is a special case of
// TriggerEvent.
func Start(ctx mctx.Context) error {
	return TriggerEvent(ctx, start)
}

// OnStop registers the given Hook to run when Stop is called. This is a special
// case of OnEvent.
func OnStop(ctx mctx.Context, hook Hook) {
	OnEvent(ctx, stop, hook)
}

// Stop runs all Hooks registered using OnStop. This is a special case of
// TriggerEvent.
func Stop(ctx mctx.Context) error {
	return TriggerEvent(ctx, stop)
}
