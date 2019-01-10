// Package mctx extends the builtin context package to organize Contexts into a
// hierarchy. Each node in the hierarchy is given a name and is aware of all of
// its ancestors.
//
// This package also provides extra functionality which allows contexts
// to be more useful when used in the hierarchy.
//
// All functions and methods in this package are thread-safe unless otherwise
// noted.
package mctx

import (
	"context"
	"sync"
	"time"
)

// Context is the same as the builtin type, but is used to indicate that the
// Context originally came from this package (aka New or ChildOf).
type Context context.Context

// CancelFunc is a direct alias of the type from the context package, see its
// docs.
type CancelFunc = context.CancelFunc

// WithValue mimics the function from the context package.
func WithValue(parent Context, key, val interface{}) Context {
	return Context(context.WithValue(context.Context(parent), key, val))
}

// WithCancel mimics the function from the context package.
func WithCancel(parent Context) (Context, CancelFunc) {
	ctx, fn := context.WithCancel(context.Context(parent))
	return Context(ctx), fn
}

// WithDeadline mimics the function from the context package.
func WithDeadline(parent Context, t time.Time) (Context, CancelFunc) {
	ctx, fn := context.WithDeadline(context.Context(parent), t)
	return Context(ctx), fn
}

// WithTimeout mimics the function from the context package.
func WithTimeout(parent Context, d time.Duration) (Context, CancelFunc) {
	ctx, fn := context.WithTimeout(context.Context(parent), d)
	return Context(ctx), fn
}

////////////////////////////////////////////////////////////////////////////////

type ctxKey int

type mutVal struct {
	l sync.RWMutex
	v interface{}
}

type ctxState struct {
	path     []string
	l        sync.RWMutex
	parent   Context
	children map[string]Context

	mutL    sync.RWMutex
	mutVals map[interface{}]*mutVal
}

func getCtxState(ctx Context) *ctxState {
	s, _ := ctx.Value(ctxKey(0)).(*ctxState)
	if s == nil {
		panic("non-conforming context used")
	}
	return s
}

func withCtxState(ctx Context, s *ctxState) Context {
	return WithValue(ctx, ctxKey(0), s)
}

// New returns a new context which can be used as the root context for all
// purposes in this framework.
func New() Context {
	return withCtxState(Context(context.Background()), &ctxState{})
}

// Path returns the sequence of names which were used to produce this context
// via the ChildOf function.
func Path(ctx Context) []string {
	return getCtxState(ctx).path
}

// Children returns all children of this context which have been created by
// ChildOf, mapped by their name.
func Children(ctx Context) map[string]Context {
	s := getCtxState(ctx)
	out := map[string]Context{}
	s.l.RLock()
	defer s.l.RUnlock()
	for name, childCtx := range s.children {
		out[name] = childCtx
	}
	return out
}

// Parent returns the parent Context of the given one, or nil if this is a root
// context (i.e. returned from New).
func Parent(ctx Context) Context {
	return getCtxState(ctx).parent
}

// Root returns the root Context from which this Context and all of its parents
// were derived (i.e. the Context which was originally returned from New).
//
// If the given Context is the root then it is returned as-id.
func Root(ctx Context) Context {
	for {
		s := getCtxState(ctx)
		if s.parent == nil {
			return ctx
		}
		ctx = s.parent
	}
}

// ChildOf creates a child of the given context with the given name and returns
// it. The Path of the returned context will be the path of the parent with its
// name appended to it. The Children function can be called on the parent to
// retrieve all children which have been made using this function.
//
// TODO If the given Context already has a child with the given name that child
// will be returned.
func ChildOf(ctx Context, name string) Context {
	s, childS := getCtxState(ctx), new(ctxState)

	s.l.Lock()
	defer s.l.Unlock()

	// set child's path field
	childS.path = make([]string, 0, len(s.path)+1)
	childS.path = append(childS.path, s.path...)
	childS.path = append(childS.path, name)

	// set child's parent field
	childS.parent = ctx

	// create child's ctx and store it in parent
	childCtx := withCtxState(ctx, childS)
	if s.children == nil {
		s.children = map[string]Context{}
	}
	s.children[name] = childCtx
	return childCtx
}

////////////////////////////////////////////////////////////////////////////////
// code related to mutable values

// MutableValue acts like the Value method, except that it only deals with
// keys/values set by SetMutableValue.
func MutableValue(ctx Context, key interface{}) interface{} {
	s := getCtxState(ctx)
	s.mutL.RLock()
	defer s.mutL.RUnlock()
	if s.mutVals == nil {
		return nil
	} else if mVal, ok := s.mutVals[key]; ok {
		mVal.l.RLock()
		defer mVal.l.RUnlock()
		return mVal.v
	}
	return nil
}

// GetSetMutableValue is used to interact with a mutable value on the context in
// a thread-safe way. The key's value is retrieved and passed to the callback.
// The value returned from the callback is stored back into the context as well
// as being returned from this function.
//
// If noCallbackIfSet is set to true, then if the key is already set the value
// will be returned without calling the callback.
//
// The callback returning nil is equivalent to unsetting the value.
//
// Children of this context will _not_ inherit any of its mutable values.
//
// Within the callback it is fine to call other functions/methods on the
// Context, except for those related to mutable values for this same key (e.g.
// MutableValue and SetMutableValue).
func GetSetMutableValue(
	ctx Context, noCallbackIfSet bool,
	key interface{}, fn func(interface{}) interface{},
) interface{} {
	s := getCtxState(ctx)

	// if noCallbackIfSet, do a fast lookup with MutableValue first.
	if noCallbackIfSet {
		if v := MutableValue(ctx, key); v != nil {
			return v
		}
	}

	s.mutL.Lock()
	if s.mutVals == nil {
		s.mutVals = map[interface{}]*mutVal{}
	}
	mVal, ok := s.mutVals[key]
	if !ok {
		mVal = new(mutVal)
		s.mutVals[key] = mVal
	}
	s.mutL.Unlock()

	mVal.l.Lock()
	defer mVal.l.Unlock()

	// It's possible something happened between the first check inside the
	// read-lock and now, so double check this case. It's still good to have the
	// read-lock check there, it'll handle 99% of the cases.
	if noCallbackIfSet && mVal.v != nil {
		return mVal.v
	}

	mVal.v = fn(mVal.v)

	// TODO if the new v is nil then key could be deleted out of mutVals. But
	// doing so would be weird in the case that there's another routine which
	// has already pulled this same mVal out of mutVals and is waiting on its
	// mutex.
	return mVal.v
}
