package mctx

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestInheritance(t *T) {
	ctx := New()
	ctx1 := ChildOf(ctx, "1")
	ctx1a := ChildOf(ctx1, "a")
	ctx1b := ChildOf(ctx1, "b")
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
		massert.Equal(Root(ctx), ctx),
		massert.Equal(Root(ctx1), ctx),
		massert.Equal(Root(ctx1a), ctx),
		massert.Equal(Root(ctx1b), ctx),
		massert.Equal(Root(ctx2), ctx),
	))
}

func TestMutableValues(t *T) {
	fn := func(v interface{}) interface{} {
		if v == nil {
			return 0
		}
		return v.(int) + 1
	}

	var aa []massert.Assertion

	ctx := New()
	aa = append(aa, massert.Equal(GetSetMutableValue(ctx, false, 0, fn), 0))
	aa = append(aa, massert.Equal(GetSetMutableValue(ctx, false, 0, fn), 1))
	aa = append(aa, massert.Equal(GetSetMutableValue(ctx, true, 0, fn), 1))

	aa = append(aa, massert.Equal(MutableValue(ctx, 0), 1))

	ctx1 := ChildOf(ctx, "one")
	aa = append(aa, massert.Equal(GetSetMutableValue(ctx1, true, 0, fn), 0))
	aa = append(aa, massert.Equal(GetSetMutableValue(ctx1, false, 0, fn), 1))
	aa = append(aa, massert.Equal(GetSetMutableValue(ctx1, true, 0, fn), 1))

	massert.Fatal(t, massert.All(aa...))
}
