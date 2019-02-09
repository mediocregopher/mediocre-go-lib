package merr

import (
	"context"
	"errors"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestError(t *T) {
	e := New(context.Background(), "foo",
		"a", "aaa aaa\n",
		"c", "ccc\nccc\n",
		"d\t", "weird key but ok",
	)
	exp := `foo
	* a: aaa aaa
	* c: 
		ccc
		ccc
	* d: weird key but ok
	* errLoc: merr/merr_test.go:13`
	massert.Fatal(t, massert.Equal(exp, e.Error()))
}

func TestBase(t *T) {
	errFoo, errBar := errors.New("foo"), errors.New("bar")
	erFoo := Wrap(context.Background(), errFoo)
	massert.Fatal(t, massert.All(
		massert.Nil(Base(nil)),
		massert.Equal(errFoo, Base(erFoo)),
		massert.Equal(errBar, Base(errBar)),
		massert.Not(massert.Equal(errFoo, erFoo)),
		massert.Not(massert.Equal(errBar, Base(erFoo))),
		massert.Equal(true, Equal(errFoo, erFoo)),
		massert.Equal(false, Equal(errBar, erFoo)),
	))
}

func TestValue(t *T) {
	massert.Fatal(t, massert.All(
		massert.Nil(WithValue(nil, "foo", "bar")),
		massert.Nil(Value(nil, "foo")),
	))

	e1 := New(context.Background(), "foo")
	e1 = WithValue(e1, "a", "A")
	e2 := WithValue(errors.New("bar"), "a", "A")
	massert.Fatal(t, massert.All(
		massert.Equal("A", Value(e1, "a")),
		massert.Equal("A", Value(e2, "a")),
	))

	e3 := WithValue(e2, "a", "AAA")
	massert.Fatal(t, massert.All(
		massert.Equal("A", Value(e1, "a")),
		massert.Equal("A", Value(e2, "a")),
		massert.Equal("AAA", Value(e3, "a")),
	))
}

func mkErr(ctx context.Context, err error) error {
	return Wrap(ctx, err) // it's important that this is line 65
}

func TestCtx(t *T) {
	ctxA := mctx.Annotate(context.Background(), "0", "ZERO", "1", "one")
	ctxB := mctx.Annotate(context.Background(), "1", "ONE", "2", "TWO")

	// use mkErr so that it's easy to test that the stack info isn't overwritten
	// when Wrap is called with ctxB.
	e := mkErr(ctxA, errors.New("hello"))
	e = Wrap(ctxB, e)

	err := massert.Equal(map[string]string{
		"0":      "ZERO",
		"1":      "ONE",
		"2":      "TWO",
		"err":    "hello",
		"errLoc": "merr/merr_test.go:65",
	}, mctx.Annotations(Context(e)).StringMap()).Assert()
	if err != nil {
		t.Fatal(err)
	}
}
