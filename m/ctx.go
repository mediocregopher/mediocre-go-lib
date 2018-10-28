package m

import (
	"context"
	"sync"
)

type ctxKey int

const (
	ctxStateKey ctxKey = iota
)

type ctxState struct {
	path     []string
	l        sync.RWMutex
	children map[string]context.Context
}

func newCtxState() *ctxState {
	return &ctxState{
		children: map[string]context.Context{},
	}
}

// Ctx returns a new context which can be used as the root context for all
// purposes in this framework.
func Ctx() context.Context {
	return context.WithValue(context.Background(), ctxStateKey, newCtxState())
}

func getCtxState(ctx context.Context) *ctxState {
	s, _ := ctx.Value(ctxStateKey).(*ctxState)
	if s == nil {
		panic("non-conforming context used")
	}
	return s
}

// Path returns the sequence of names which were used to produce this context
// via the ChildOf function.
func Path(ctx context.Context) []string {
	return getCtxState(ctx).path
}

// Children returns all children of this context which have been created by
// ChildOf, mapped by their name.
func Children(ctx context.Context) map[string]context.Context {
	s := getCtxState(ctx)
	out := map[string]context.Context{}
	s.l.RLock()
	defer s.l.RUnlock()
	for name, childCtx := range s.children {
		out[name] = childCtx
	}
	return out
}

// ChildOf creates a child of the given context with the given name and returns
// it. The Path of the returned context will be the path of the parent with its
// name appended to it. The Children function can be called on the parent to
// retrieve all children which have been made using this function.
func ChildOf(ctx context.Context, name string) context.Context {
	s, childS := getCtxState(ctx), newCtxState()
	s.l.Lock()
	defer s.l.Unlock()
	childS.path = make([]string, 0, len(s.path)+1)
	childS.path = append(childS.path, s.path...)
	childS.path = append(childS.path, name)
	childCtx := context.WithValue(ctx, ctxStateKey, childS)
	s.children[name] = childCtx
	return childCtx
}
