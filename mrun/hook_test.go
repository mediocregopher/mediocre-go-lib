package mrun

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestHooks(t *T) {
	var out []int
	mkHook := func(i int) Hook {
		return func(context.Context) error {
			out = append(out, i)
			return nil
		}
	}

	ctx := context.Background()
	ctx = RegisterHook(ctx, 0, mkHook(1))
	ctx = RegisterHook(ctx, 0, mkHook(2))

	ctxA := mctx.NewChild(ctx, "a")
	ctxA = RegisterHook(ctxA, 0, mkHook(3))
	ctxA = RegisterHook(ctxA, 999, mkHook(999)) // different key
	ctx = mctx.WithChild(ctx, ctxA)

	ctx = RegisterHook(ctx, 0, mkHook(4))

	ctxB := mctx.NewChild(ctx, "b")
	ctxB = RegisterHook(ctxB, 0, mkHook(5))
	ctxB1 := mctx.NewChild(ctxB, "1")
	ctxB1 = RegisterHook(ctxB1, 0, mkHook(6))
	ctxB = mctx.WithChild(ctxB, ctxB1)
	ctx = mctx.WithChild(ctx, ctxB)

	massert.Fatal(t, massert.All(
		massert.Nil(TriggerHooks(ctx, 0)),
		massert.Equal([]int{1, 2, 3, 4, 5, 6}, out),
	))

	out = nil
	massert.Fatal(t, massert.All(
		massert.Nil(TriggerHooksReverse(ctx, 0)),
		massert.Equal([]int{6, 5, 4, 3, 2, 1}, out),
	))
}
