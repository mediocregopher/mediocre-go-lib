package mrun

import (
	"errors"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestHooks(t *T) {
	ch := make(chan int, 10)
	ctx := mctx.New()
	ctxChild := mctx.ChildOf(ctx, "child")

	mkHook := func(i int) Hook {
		return func(mctx.Context) error {
			ch <- i
			return nil
		}
	}

	RegisterHook(ctx, 0, mkHook(0))
	RegisterHook(ctxChild, 0, mkHook(1))
	RegisterHook(ctx, 0, mkHook(2))

	bogusErr := errors.New("bogus error")
	RegisterHook(ctxChild, 0, func(mctx.Context) error { return bogusErr })

	RegisterHook(ctx, 0, mkHook(3))
	RegisterHook(ctx, 0, mkHook(4))

	massert.Fatal(t, massert.All(
		massert.Equal(bogusErr, TriggerHooks(ctx, 0)),
		massert.Equal(0, <-ch),
		massert.Equal(1, <-ch),
		massert.Equal(2, <-ch),
	))

	// after the error the 3 and 4 Hooks should still be registered, but not
	// called yet.

	select {
	case <-ch:
		t.Fatal("Hooks should not have been called yet")
	default:
	}

	massert.Fatal(t, massert.All(
		massert.Nil(TriggerHooks(ctx, 0)),
		massert.Equal(3, <-ch),
		massert.Equal(4, <-ch),
	))
}
