package mrun

import (
	"context"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
)

// Hook describes a function which can be registered to trigger on an event via
// the WithHook function.
type Hook func(context.Context) error

type hookKey struct {
	key interface{}
}

// AddHook registers a Hook under a typed key. The Hook will be called when
// TriggerHooks is called with that same key. Multiple Hooks can be registered
// for the same key, and will be called sequentially when triggered.
//
// Hooks will be called with whatever Context is passed into TriggerHooks.
func AddHook(cmp *mcmp.Component, key interface{}, hook Hook) {
	mcmp.AddSeriesValue(cmp, hookKey{key}, hook)
}

func triggerHooks(
	ctx context.Context,
	cmp *mcmp.Component,
	key interface{},
	next func([]mcmp.SeriesElement) (mcmp.SeriesElement, []mcmp.SeriesElement),
) error {
	els := mcmp.SeriesElements(cmp, hookKey{key})
	var el mcmp.SeriesElement
	for {
		if len(els) == 0 {
			break
		}
		el, els = next(els)
		if el.Child != nil {
			if err := triggerHooks(ctx, el.Child, key, next); err != nil {
				return err
			}
			continue
		}
		hook := el.Value.(Hook)
		if err := hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

// TriggerHooks causes all Hooks registered with AddHook on the Component under
// the given key to be called in the order they were registered. The given
// Context is passed into all Hooks being called.
//
// If any Hook returns an error no further Hooks will be called and that error
// will be returned.
//
// If the Component has children (see the mcmp package), and those children have
// Hooks registered under this key, then their Hooks will be called in the
// expected order. See package docs for an example.
func TriggerHooks(
	ctx context.Context,
	cmp *mcmp.Component,
	key interface{},
) error {
	next := func(els []mcmp.SeriesElement) (
		mcmp.SeriesElement, []mcmp.SeriesElement,
	) {
		return els[0], els[1:]
	}
	return triggerHooks(ctx, cmp, key, next)
}

// TriggerHooksReverse is the same as TriggerHooks except that registered Hooks
// are called in the reverse order in which they were registered.
func TriggerHooksReverse(ctx context.Context, cmp *mcmp.Component, key interface{}) error {
	next := func(els []mcmp.SeriesElement) (
		mcmp.SeriesElement, []mcmp.SeriesElement,
	) {
		last := len(els) - 1
		return els[last], els[:last]
	}
	return triggerHooks(ctx, cmp, key, next)
}

type builtinEvent int

const (
	initEvent builtinEvent = iota
	shutdownEvent
)

// InitHook registers the given Hook to run when Init is called. This is a
// special case of AddHook.
//
// As a convention Hooks running on the init event should block only as long as
// it takes to ensure that whatever is running can do so successfully. For
// short-lived tasks this isn't a problem, but long-lived tasks (e.g. a web
// server) will want to use the Hook only to initialize, and spawn off a
// go-routine to do their actual work. Long-lived tasks should set themselves up
// to shutdown on the shutdown event (see ShutdownHook).
func InitHook(cmp *mcmp.Component, hook Hook) {
	AddHook(cmp, initEvent, hook)
}

// Init runs all Hooks registered using InitHook. This is a special case of
// TriggerHooks.
func Init(ctx context.Context, cmp *mcmp.Component) error {
	return TriggerHooks(ctx, cmp, initEvent)
}

// ShutdownHook registers the given Hook to run when Shutdown is called. This is
// a special case of AddHook.
//
// See InitHook for more on the relationship between Init(Hook) and
// Shutdown(Hook).
func ShutdownHook(cmp *mcmp.Component, hook Hook) {
	AddHook(cmp, shutdownEvent, hook)
}

// Shutdown runs all Hooks registered using ShutdownHook in the reverse order in
// which they were registered. This is a special case of TriggerHooks.
func Shutdown(ctx context.Context, cmp *mcmp.Component) error {
	return TriggerHooksReverse(ctx, cmp, shutdownEvent)
}
