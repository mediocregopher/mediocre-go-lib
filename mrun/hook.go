package mrun

import "github.com/mediocregopher/mediocre-go-lib/mctx"

type ctxEventKeyWrap struct {
	key interface{}
}

// Hook describes a function which can be registered to trigger on an event via
// the RegisterHook function.
type Hook func(mctx.Context) error

// RegisterHook registers a Hook under a typed key. The Hook will be called when
// TriggerHooks is called with that same key. Multiple Hooks can be registered
// for the same key, and will be called sequentially when triggered.
//
// RegisterHook registers Hooks onto the root of the given Context. Therefore,
// Hooks will be triggered in the global order they were registered. For
// example: if one Hook is registered on a Context, then one is registered on a
// child of that Context, then another one is registered on the original Context
// again, the three Hooks will be triggered in the order: parent, child,
// parent.
//
// Hooks will be called with whatever Context is passed into TriggerHooks.
func RegisterHook(ctx mctx.Context, key interface{}, hook Hook) {
	ctx = mctx.Root(ctx)
	mctx.GetSetMutableValue(ctx, false, ctxEventKeyWrap{key}, func(v interface{}) interface{} {
		hooks, _ := v.([]Hook)
		return append(hooks, hook)
	})
}

func triggerHooks(ctx mctx.Context, key interface{}, next func([]Hook) (Hook, []Hook)) error {
	rootCtx := mctx.Root(ctx)
	var err error
	mctx.GetSetMutableValue(rootCtx, false, ctxEventKeyWrap{key}, func(i interface{}) interface{} {
		var hook Hook
		hooks, _ := i.([]Hook)
		for {
			if len(hooks) == 0 {
				break
			}
			hook, hooks = next(hooks)

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

// TriggerHooks causes all Hooks registered with RegisterHook under the given
// key to be called in the global order they were registered, using the given
// Context as their input parameter. The given Context does not need to be the
// root Context (see RegisterHook).
//
// If any Hook returns an error no further Hooks will be called and that error
// will be returned.
//
// TriggerHooks causes all Hooks which were called to be de-registered. If an
// error caused execution to stop prematurely then any Hooks which were not
// called will remain registered.
func TriggerHooks(ctx mctx.Context, key interface{}) error {
	return triggerHooks(ctx, key, func(hooks []Hook) (Hook, []Hook) {
		return hooks[0], hooks[1:]
	})
}

// TriggerHooksReverse is the same as TriggerHooks except that registered Hooks
// are called in the reverse order in which they were registered.
func TriggerHooksReverse(ctx mctx.Context, key interface{}) error {
	return triggerHooks(ctx, key, func(hooks []Hook) (Hook, []Hook) {
		last := len(hooks) - 1
		return hooks[last], hooks[:last]
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
func OnStart(ctx mctx.Context, hook Hook) {
	RegisterHook(ctx, start, hook)
}

// Start runs all Hooks registered using OnStart. This is a special case of
// TriggerHooks.
func Start(ctx mctx.Context) error {
	return TriggerHooks(ctx, start)
}

// OnStop registers the given Hook to run when Stop is called. This is a special
// case of RegisterHook.
func OnStop(ctx mctx.Context, hook Hook) {
	RegisterHook(ctx, stop, hook)
}

// Stop runs all Hooks registered using OnStop in the reverse order in which
// they were registered. This is a special case of TriggerHooks.
func Stop(ctx mctx.Context) error {
	return TriggerHooksReverse(ctx, stop)
}
