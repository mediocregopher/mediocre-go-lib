package mrun

import (
	"context"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
)

// Hook describes a function which can be registered to trigger on an event via
// the RegisterHook function.
type Hook func(context.Context) error

type ctxKey int

const (
	ctxKeyHookEls ctxKey = iota
	ctxKeyNumChildren
	ctxKeyNumHooks
)

type ctxKeyWrap struct {
	key     ctxKey
	userKey interface{}
}

// because we want Hooks to be called in the order created, taking into account
// the creation of children and their hooks as well, we create a sequence of
// elements which can either be a Hook or a child.
type hookEl struct {
	hook  Hook
	child context.Context
}

func ctxKeys(userKey interface{}) (ctxKeyWrap, ctxKeyWrap, ctxKeyWrap) {
	return ctxKeyWrap{
			key:     ctxKeyHookEls,
			userKey: userKey,
		}, ctxKeyWrap{
			key:     ctxKeyNumChildren,
			userKey: userKey,
		}, ctxKeyWrap{
			key:     ctxKeyNumHooks,
			userKey: userKey,
		}
}

// getHookEls retrieves a copy of the []hookEl in the Context and possibly
// appends more elements if more children have been added since that []hookEl
// was created.
//
// this also returns the latest numChildren and numHooks values for convenience.
func getHookEls(ctx context.Context, userKey interface{}) ([]hookEl, int, int) {
	hookElsKey, numChildrenKey, numHooksKey := ctxKeys(userKey)
	lastNumChildren, _ := mctx.LocalValue(ctx, numChildrenKey).(int)
	lastNumHooks, _ := mctx.LocalValue(ctx, numHooksKey).(int)
	lastHookEls, _ := mctx.LocalValue(ctx, hookElsKey).([]hookEl)
	children := mctx.Children(ctx)

	// plus 1 in case we wanna append something else outside this function
	hookEls := make([]hookEl, len(lastHookEls), lastNumHooks+len(children)-lastNumChildren+1)
	copy(hookEls, lastHookEls)
	for _, child := range children[lastNumChildren:] {
		hookEls = append(hookEls, hookEl{child: child})
	}
	return hookEls, len(children), lastNumHooks
}

// RegisterHook registers a Hook under a typed key. The Hook will be called when
// TriggerHooks is called with that same key. Multiple Hooks can be registered
// for the same key, and will be called sequentially when triggered.
//
// Hooks will be called with whatever Context is passed into TriggerHooks.
func RegisterHook(ctx context.Context, key interface{}, hook Hook) context.Context {
	hookEls, numChildren, numHooks := getHookEls(ctx, key)
	hookEls = append(hookEls, hookEl{hook: hook})

	hookElsKey, numChildrenKey, numHooksKey := ctxKeys(key)
	ctx = mctx.WithLocalValue(ctx, hookElsKey, hookEls)
	ctx = mctx.WithLocalValue(ctx, numChildrenKey, numChildren)
	ctx = mctx.WithLocalValue(ctx, numHooksKey, numHooks+1)
	return ctx
}

func triggerHooks(ctx context.Context, userKey interface{}, next func([]hookEl) (hookEl, []hookEl)) error {
	hookEls, _, _ := getHookEls(ctx, userKey)
	var hookEl hookEl
	for {
		if len(hookEls) == 0 {
			break
		}
		hookEl, hookEls = next(hookEls)
		if hookEl.child != nil {
			if err := triggerHooks(hookEl.child, userKey, next); err != nil {
				return err
			}
		} else if err := hookEl.hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

// TriggerHooks causes all Hooks registered with RegisterHook on the Context
// (and its predecessors) under the given key to be called in the order they
// were registered.
//
// If any Hook returns an error no further Hooks will be called and that error
// will be returned.
//
// If the Context has children (see the mctx package), and those children have
// Hooks registered under this key, then their Hooks will be called in the
// expected order. For example:
//
//	// parent context has hookA registered
//	ctx := context.Background()
//	ctx = RegisterHook(ctx, 0, hookA)
//
//	// child context has hookB registered
//	childCtx := mctx.NewChild(ctx, "child")
//	childCtx = RegisterHook(childCtx, 0, hookB)
//	ctx = mctx.WithChild(ctx, childCtx) // needed to link childCtx to ctx
//
//	// parent context has another Hook, hookC, registered
//	ctx = RegisterHook(ctx, 0, hookC)
//
//	// The Hooks will be triggered in the order: hookA, hookB, then hookC
//	err := TriggerHooks(ctx, 0)
//
func TriggerHooks(ctx context.Context, key interface{}) error {
	return triggerHooks(ctx, key, func(hookEls []hookEl) (hookEl, []hookEl) {
		return hookEls[0], hookEls[1:]
	})
}

// TriggerHooksReverse is the same as TriggerHooks except that registered Hooks
// are called in the reverse order in which they were registered.
func TriggerHooksReverse(ctx context.Context, key interface{}) error {
	return triggerHooks(ctx, key, func(hookEls []hookEl) (hookEl, []hookEl) {
		last := len(hookEls) - 1
		return hookEls[last], hookEls[:last]
	})
}

type builtinEvent int

const (
	start builtinEvent = iota
	stop
)

// OnStart registers the given Hook to run when Start is called. This is a
// special case of RegisterHook.
//
// As a convention Hooks running on the start event should block only as long as
// it takes to ensure that whatever is running can do so successfully. For
// short-lived tasks this isn't a problem, but long-lived tasks (e.g. a web
// server) will want to use the Hook only to initialize, and spawn off a
// go-routine to do their actual work. Long-lived tasks should set themselves up
// to stop on the stop event (see OnStop).
func OnStart(ctx context.Context, hook Hook) context.Context {
	return RegisterHook(ctx, start, hook)
}

// Start runs all Hooks registered using OnStart. This is a special case of
// TriggerHooks.
func Start(ctx context.Context) error {
	return TriggerHooks(ctx, start)
}

// OnStop registers the given Hook to run when Stop is called. This is a special
// case of RegisterHook.
func OnStop(ctx context.Context, hook Hook) context.Context {
	return RegisterHook(ctx, stop, hook)
}

// Stop runs all Hooks registered using OnStop in the reverse order in which
// they were registered. This is a special case of TriggerHooks.
func Stop(ctx context.Context) error {
	return TriggerHooksReverse(ctx, stop)
}
