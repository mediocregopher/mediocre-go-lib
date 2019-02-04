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
	"sync"
	"time"

	goctx "context"
)

// Context is the same as the builtin type, but is used to indicate that the
// Context originally came from this package (aka New or ChildOf).
type Context goctx.Context

// CancelFunc is a direct alias of the type from the context package, see its
// docs.
type CancelFunc = goctx.CancelFunc

// WithValue mimics the function from the context package.
func WithValue(parent Context, key, val interface{}) Context {
	return Context(goctx.WithValue(goctx.Context(parent), key, val))
}

// WithCancel mimics the function from the context package.
func WithCancel(parent Context) (Context, CancelFunc) {
	ctx, fn := goctx.WithCancel(goctx.Context(parent))
	return Context(ctx), fn
}

// WithDeadline mimics the function from the context package.
func WithDeadline(parent Context, t time.Time) (Context, CancelFunc) {
	ctx, fn := goctx.WithDeadline(goctx.Context(parent), t)
	return Context(ctx), fn
}

// WithTimeout mimics the function from the context package.
func WithTimeout(parent Context, d time.Duration) (Context, CancelFunc) {
	ctx, fn := goctx.WithTimeout(goctx.Context(parent), d)
	return Context(ctx), fn
}

////////////////////////////////////////////////////////////////////////////////

type mutVal struct {
	l sync.RWMutex
	v interface{}
}

type context struct {
	goctx.Context

	path     []string
	l        sync.RWMutex
	parent   *context
	children map[string]Context

	mutL    sync.RWMutex
	mutVals map[interface{}]*mutVal
}

// New returns a new context which can be used as the root context for all
// purposes in this framework.
func New() Context {
	return &context{Context: goctx.Background()}
}

func getCtx(Ctx Context) *context {
	ctx, ok := Ctx.(*context)
	if !ok {
		panic("non-conforming Context used")
	}
	return ctx
}

// Path returns the sequence of names which were used to produce this context
// via the ChildOf function.
func Path(Ctx Context) []string {
	return getCtx(Ctx).path
}

// Children returns all children of this context which have been created by
// ChildOf, mapped by their name.
func Children(Ctx Context) map[string]Context {
	ctx := getCtx(Ctx)
	out := map[string]Context{}
	ctx.l.RLock()
	defer ctx.l.RUnlock()
	for name, childCtx := range ctx.children {
		out[name] = childCtx
	}
	return out
}

// Parent returns the parent Context of the given one, or nil if this is a root
// context (i.e. returned from New).
func Parent(Ctx Context) Context {
	return getCtx(Ctx).parent
}

// Root returns the root Context from which this Context and all of its parents
// were derived (i.e. the Context which was originally returned from New).
//
// If the given Context is the root then it is returned as-id.
func Root(Ctx Context) Context {
	ctx := getCtx(Ctx)
	for {
		if ctx.parent == nil {
			return ctx
		}
		ctx = ctx.parent
	}
}

// ChildOf creates a child of the given context with the given name and returns
// it. The Path of the returned context will be the path of the parent with its
// name appended to it. The Children function can be called on the parent to
// retrieve all children which have been made using this function.
//
// TODO If the given Context already has a child with the given name that child
// will be returned.
func ChildOf(Ctx Context, name string) Context {
	ctx, childCtx := getCtx(Ctx), new(context)

	ctx.l.Lock()
	defer ctx.l.Unlock()

	// set child's path field
	childCtx.path = make([]string, 0, len(ctx.path)+1)
	childCtx.path = append(childCtx.path, ctx.path...)
	childCtx.path = append(childCtx.path, name)

	// set child's parent field
	childCtx.parent = ctx

	// create child's ctx and store it in parent
	if ctx.children == nil {
		ctx.children = map[string]Context{}
	}
	ctx.children[name] = childCtx
	return childCtx
}

// BreadthFirstVisit visits this Context and all of its children, and their
// children, in a breadth-first order. If the callback returns false then the
// function returns without visiting any more Contexts.
//
// The exact order of visitation is non-deterministic.
func BreadthFirstVisit(Ctx Context, callback func(Context) bool) {
	queue := []Context{Ctx}
	for len(queue) > 0 {
		if !callback(queue[0]) {
			return
		}
		for _, child := range Children(queue[0]) {
			queue = append(queue, child)
		}
		queue = queue[1:]
	}
}

////////////////////////////////////////////////////////////////////////////////
// code related to mutable values

// MutableValue acts like the Value method, except that it only deals with
// keys/values set by SetMutableValue.
func MutableValue(Ctx Context, key interface{}) interface{} {
	ctx := getCtx(Ctx)
	ctx.mutL.RLock()
	defer ctx.mutL.RUnlock()
	if ctx.mutVals == nil {
		return nil
	} else if mVal, ok := ctx.mutVals[key]; ok {
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
	Ctx Context, noCallbackIfSet bool,
	key interface{}, fn func(interface{}) interface{},
) interface{} {

	// if noCallbackIfSet, do a fast lookup with MutableValue first.
	if noCallbackIfSet {
		if v := MutableValue(Ctx, key); v != nil {
			return v
		}
	}

	ctx := getCtx(Ctx)
	ctx.mutL.Lock()
	if ctx.mutVals == nil {
		ctx.mutVals = map[interface{}]*mutVal{}
	}
	mVal, ok := ctx.mutVals[key]
	if !ok {
		mVal = new(mutVal)
		ctx.mutVals[key] = mVal
	}
	ctx.mutL.Unlock()

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
