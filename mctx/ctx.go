// Package mctx provides a framework, based around the Context type, for
// managing configuration, initialization, runtime, and shutdown of a binary.
// The framework allows components to manage these aspects individually by means
// of a heirarchical system.
//
// Each node in the hierarchy is given a name and is aware of all of its
// ancestors, and can incorporate this information into its functionality.
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

type ctxState struct {
	path     []string
	l        sync.RWMutex
	parent   Context
	children map[string]Context

	mutVals map[interface{}]interface{}
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
	return withCtxState(Context(context.Background()), &ctxState{
		//logger: mlog.NewLogger(os.Stderr),
	})
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

// ChildOf creates a child of the given context with the given name and returns
// it. The Path of the returned context will be the path of the parent with its
// name appended to it. The Children function can be called on the parent to
// retrieve all children which have been made using this function.
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

	// set up child's logger
	//pathStr := "/"
	//if len(childS.path) > 0 {
	//	pathStr += path.Join(childS.path...)
	//}
	//childS.logger = s.logger.Clone()
	//childS.logger.SetWriteFn(func(w io.Writer, msg mlog.Message) error {
	//	msg.Msg = "(" + pathStr + ") " + msg.Msg
	//	return mlog.DefaultWriteFn(w, msg)
	//})

	// create child's ctx and store it in parent
	childCtx := withCtxState(ctx, childS)
	if s.children == nil {
		s.children = map[string]Context{}
	}
	s.children[name] = childCtx
	return childCtx
}

// TODO these might not be worth the effort, they're very subject to
// race-conditions

// MutableValue acts like the Value method, except that it only deals with
// keys/values set by SetMutableValue.
func MutableValue(ctx Context, key interface{}) interface{} {
	s := getCtxState(ctx)
	s.l.RLock()
	defer s.l.RUnlock()
	if s.mutVals == nil {
		return nil
	}
	return s.mutVals[key]
}

// GetSetMutableValue is used to interact with a mutable value on the context in
// a thread-safe way. The key's value is retrieved and passed to the callback.
// The value returned from the callback is stored to back into the context as
// well as being returned from this function.
//
// If noCallbackIfSet is set to true, then this function will only return the
// value for the key if it's already set and not use the callback at all.
//
// The callback returning nil is equivalent to unsetting the value.
//
// Children of this context will _not_ inherit any of its mutable values.
func GetSetMutableValue(
	ctx Context, noCallbackIfSet bool,
	key interface{}, fn func(interface{}) interface{},
) interface{} {
	s := getCtxState(ctx)

	if noCallbackIfSet {
		s.l.RLock()
		if s.mutVals != nil && s.mutVals[key] != nil {
			defer s.l.RUnlock()
			return s.mutVals[key]
		}
		s.l.RUnlock()
	}

	s.l.Lock()
	defer s.l.Unlock()

	if s.mutVals == nil {
		s.mutVals = map[interface{}]interface{}{}
	}
	val := s.mutVals[key]

	// It's possible something happened between the first check inside the
	// read-lock and now, so double check this case. It's still good to have the
	// read-lock check there, it'll handle 99% of the cases.
	if noCallbackIfSet && val != nil {
		return val
	}

	val = fn(val)
	if val == nil {
		delete(s.mutVals, key)
	} else {
		s.mutVals[key] = val
	}
	return val
}
