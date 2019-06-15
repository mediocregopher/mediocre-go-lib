package mrun

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
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

	cmp := new(mcmp.Component)
	AddHook(cmp, 0, mkHook(1))
	AddHook(cmp, 0, mkHook(2))

	cmpA := cmp.Child("a")
	AddHook(cmpA, 0, mkHook(3))
	AddHook(cmpA, 999, mkHook(999)) // different key

	AddHook(cmp, 0, mkHook(4))

	cmpB := cmp.Child("b")
	AddHook(cmpB, 0, mkHook(5))
	cmpB1 := cmpB.Child("1")
	AddHook(cmpB1, 0, mkHook(6))

	AddHook(cmp, 0, mkHook(7))

	massert.Require(t,
		massert.Nil(TriggerHooks(context.Background(), cmp, 0)),
		massert.Equal([]int{1, 2, 3, 4, 5, 6, 7}, out),
	)

	out = nil
	massert.Require(t,
		massert.Nil(TriggerHooksReverse(context.Background(), cmp, 0)),
		massert.Equal([]int{7, 6, 5, 4, 3, 2, 1}, out),
	)
}
