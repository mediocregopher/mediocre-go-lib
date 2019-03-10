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
	ctx = WithHook(ctx, 0, mkHook(1))
	ctx = WithHook(ctx, 0, mkHook(2))

	ctxA := mctx.NewChild(ctx, "a")
	ctxA = WithHook(ctxA, 0, mkHook(3))
	ctxA = WithHook(ctxA, 999, mkHook(999)) // different key
	ctx = mctx.WithChild(ctx, ctxA)

	ctx = WithHook(ctx, 0, mkHook(4))

	ctxB := mctx.NewChild(ctx, "b")
	ctxB = WithHook(ctxB, 0, mkHook(5))
	ctxB1 := mctx.NewChild(ctxB, "1")
	ctxB1 = WithHook(ctxB1, 0, mkHook(6))
	ctxB = mctx.WithChild(ctxB, ctxB1)
	ctx = mctx.WithChild(ctx, ctxB)

	ctx = WithHook(ctx, 0, mkHook(7))

	massert.Require(t,
		massert.Nil(TriggerHooks(ctx, 0)),
		massert.Equal([]int{1, 2, 3, 4, 5, 6, 7}, out),
	)

	out = nil
	massert.Require(t,
		massert.Nil(TriggerHooksReverse(ctx, 0)),
		massert.Equal([]int{7, 6, 5, 4, 3, 2, 1}, out),
	)
}
