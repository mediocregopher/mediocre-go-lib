package mctx

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestInheritance(t *T) {
	ctx := context.Background()
	ctx1 := NewChild(ctx, "1")
	ctx1a := NewChild(ctx1, "a")
	ctx1b := NewChild(ctx1, "b")
	ctx1 = WithChild(ctx1, ctx1a)
	ctx1 = WithChild(ctx1, ctx1b)
	ctx2 := NewChild(ctx, "2")
	ctx = WithChild(ctx, ctx1)
	ctx = WithChild(ctx, ctx2)

	massert.Fatal(t, massert.All(
		massert.Len(Path(ctx), 0),
		massert.Equal(Path(ctx1), []string{"1"}),
		massert.Equal(Path(ctx1a), []string{"1", "a"}),
		massert.Equal(Path(ctx1b), []string{"1", "b"}),
		massert.Equal(Path(ctx2), []string{"2"}),
	))

	massert.Fatal(t, massert.All(
		massert.Equal([]context.Context{ctx1, ctx2}, Children(ctx)),
		massert.Equal([]context.Context{ctx1a, ctx1b}, Children(ctx1)),
		massert.Len(Children(ctx2), 0),
	))
}

func TestBreadFirstVisit(t *T) {
	ctx := context.Background()
	ctx1 := NewChild(ctx, "1")
	ctx1a := NewChild(ctx1, "a")
	ctx1b := NewChild(ctx1, "b")
	ctx1 = WithChild(ctx1, ctx1a)
	ctx1 = WithChild(ctx1, ctx1b)
	ctx2 := NewChild(ctx, "2")
	ctx = WithChild(ctx, ctx1)
	ctx = WithChild(ctx, ctx2)

	{
		got := make([]context.Context, 0, 5)
		BreadthFirstVisit(ctx, func(ctx context.Context) bool {
			got = append(got, ctx)
			return true
		})
		massert.Fatal(t,
			massert.Equal([]context.Context{ctx, ctx1, ctx2, ctx1a, ctx1b}, got),
		)
	}

	{
		got := make([]context.Context, 0, 3)
		BreadthFirstVisit(ctx, func(ctx context.Context) bool {
			if len(Path(ctx)) > 1 {
				return false
			}
			got = append(got, ctx)
			return true
		})
		massert.Fatal(t,
			massert.Equal([]context.Context{ctx, ctx1, ctx2}, got),
		)
	}
}

func TestLocalValues(t *T) {

	// test with no value set
	ctx := context.Background()
	massert.Fatal(t, massert.All(
		massert.Nil(LocalValue(ctx, "foo")),
		massert.Len(LocalValues(ctx), 0),
	))

	// test basic value retrieval
	ctx = WithLocalValue(ctx, "foo", "bar")
	massert.Fatal(t, massert.All(
		massert.Equal("bar", LocalValue(ctx, "foo")),
		massert.Equal(
			map[interface{}]interface{}{"foo": "bar"},
			LocalValues(ctx),
		),
	))

	// test that doesn't conflict with WithValue
	ctx = context.WithValue(ctx, "foo", "WithValue bar")
	massert.Fatal(t, massert.All(
		massert.Equal("bar", LocalValue(ctx, "foo")),
		massert.Equal("WithValue bar", ctx.Value("foo")),
		massert.Equal(
			map[interface{}]interface{}{"foo": "bar"},
			LocalValues(ctx),
		),
	))

	// test that child doesn't get values
	child := NewChild(ctx, "child")
	massert.Fatal(t, massert.All(
		massert.Equal("bar", LocalValue(ctx, "foo")),
		massert.Nil(LocalValue(child, "foo")),
		massert.Len(LocalValues(child), 0),
	))

	// test that values on child don't affect parent values
	child = WithLocalValue(child, "foo", "child bar")
	ctx = WithChild(ctx, child)
	massert.Fatal(t, massert.All(
		massert.Equal("bar", LocalValue(ctx, "foo")),
		massert.Equal("child bar", LocalValue(child, "foo")),
		massert.Equal(
			map[interface{}]interface{}{"foo": "bar"},
			LocalValues(ctx),
		),
		massert.Equal(
			map[interface{}]interface{}{"foo": "child bar"},
			LocalValues(child),
		),
	))

	// test that two With calls on the same context generate distinct contexts
	childA := WithLocalValue(child, "foo2", "baz")
	childB := WithLocalValue(child, "foo2", "buz")
	massert.Fatal(t, massert.All(
		massert.Equal("bar", LocalValue(ctx, "foo")),
		massert.Equal("child bar", LocalValue(child, "foo")),
		massert.Nil(LocalValue(child, "foo2")),
		massert.Equal("baz", LocalValue(childA, "foo2")),
		massert.Equal("buz", LocalValue(childB, "foo2")),
		massert.Equal(
			map[interface{}]interface{}{"foo": "child bar", "foo2": "baz"},
			LocalValues(childA),
		),
		massert.Equal(
			map[interface{}]interface{}{"foo": "child bar", "foo2": "buz"},
			LocalValues(childB),
		),
	))

	// if a value overwrites a previous one the newer one should show in
	// LocalValues
	ctx = WithLocalValue(ctx, "foo", "barbar")
	massert.Fatal(t, massert.All(
		massert.Equal("barbar", LocalValue(ctx, "foo")),
		massert.Equal(
			map[interface{}]interface{}{"foo": "barbar"},
			LocalValues(ctx),
		),
	))
}
