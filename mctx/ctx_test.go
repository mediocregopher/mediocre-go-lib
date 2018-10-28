package mctx

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestContext(t *T) {
	ctx := New()
	SetMutableValue(ctx, "one", 1)

	ctx1 := ChildOf(ctx, "1")
	ctx1a := ChildOf(ctx1, "a")
	SetMutableValue(ctx1, "one", 2)
	ctx1b := ChildOf(ctx1, "b")
	SetMutableValue(ctx1b, "one", 3)
	ctx2 := ChildOf(ctx, "2")

	massert.Fatal(t, massert.All(
		massert.Len(Path(ctx), 0),
		massert.Equal(Path(ctx1), []string{"1"}),
		massert.Equal(Path(ctx1a), []string{"1", "a"}),
		massert.Equal(Path(ctx1b), []string{"1", "b"}),
		massert.Equal(Path(ctx2), []string{"2"}),
	))

	massert.Fatal(t, massert.All(
		massert.Equal(
			map[string]Context{"1": ctx1, "2": ctx2},
			Children(ctx),
		),
		massert.Equal(
			map[string]Context{"a": ctx1a, "b": ctx1b},
			Children(ctx1),
		),
		massert.Equal(
			map[string]Context{},
			Children(ctx2),
		),
	))

	massert.Fatal(t, massert.All(
		massert.Nil(Parent(ctx)),
		massert.Equal(Parent(ctx1), ctx),
		massert.Equal(Parent(ctx1a), ctx1),
		massert.Equal(Parent(ctx1b), ctx1),
		massert.Equal(Parent(ctx2), ctx),
	))

	massert.Fatal(t, massert.All(
		massert.Equal(MutableValue(ctx, "one"), 1),
		massert.Equal(MutableValue(ctx1, "one"), 2),
		massert.Equal(MutableValue(ctx1a, "one"), 1),
		massert.Equal(MutableValue(ctx1b, "one"), 3),
		massert.Equal(MutableValue(ctx2, "one"), 1),
	))
}
