package merr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/mediocregopher/mediocre-go-lib/v2/mctx"
	"github.com/mediocregopher/mediocre-go-lib/v2/mtest/massert"
)

func TestFullError(t *testing.T) {
	massert.Require(t, massert.Nil(Wrap(context.Background(), nil)))

	ctx := mctx.Annotate(context.Background(),
		"a", "aaa aaa\n",
		"c", "ccc\nccc\n",
		"d\t", "weird key but ok")

	{
		e := New(ctx, "foo")
		exp := `foo
	* a: aaa aaa
	* c: 
		ccc
		ccc
	* d: weird key but ok
	* line: merr/merr_test.go:22`
		massert.Require(t, massert.Equal(exp, e.(Error).FullError()))
	}

	{
		e := Wrap(ctx, errors.New("foo"))
		exp := `foo
	* a: aaa aaa
	* c: 
		ccc
		ccc
	* d: weird key but ok
	* line: merr/merr_test.go:34`
		massert.Require(t, massert.Equal(exp, e.(Error).FullError()))
	}
}

func TestAsIsError(t *testing.T) {
	testST := newStacktrace(0)

	ctxA := mctx.Annotate(context.Background(), "a", "1")
	ctxB := mctx.Annotate(context.Background(), "b", "2")
	errFoo := errors.New("foo")

	type test struct {
		in     error
		expAs  error
		expIs  error
		expStr string
	}

	tests := []test{
		{
			in: nil,
		},
		{
			in:     errors.New("bar"),
			expStr: "bar",
		},
		{
			in: Error{
				Err:        errFoo,
				Ctx:        ctxA,
				Stacktrace: testST,
			},
			expAs: Error{
				Err:        errFoo,
				Ctx:        ctxA,
				Stacktrace: testST,
			},
			expIs:  errFoo,
			expStr: "foo",
		},
		{
			in: fmt.Errorf("bar: %w", Error{
				Err:        errFoo,
				Ctx:        ctxA,
				Stacktrace: testST,
			}),
			expAs: Error{
				Err:        errFoo,
				Ctx:        ctxA,
				Stacktrace: testST,
			},
			expIs:  errFoo,
			expStr: "bar: foo",
		},
		{
			in: Wrap(ctxB, Error{
				Err:        errFoo,
				Ctx:        ctxA,
				Stacktrace: testST,
			}),
			expAs: Error{
				Err: Error{
					Err:        errFoo,
					Ctx:        ctxA,
					Stacktrace: testST,
				},
				Ctx:        mctx.MergeAnnotations(ctxA, ctxB),
				Stacktrace: testST,
			},
			expIs:  errFoo,
			expStr: "foo",
		},
		{
			in: Wrap(ctxB, fmt.Errorf("bar: %w", Error{
				Err:        errFoo,
				Ctx:        ctxA,
				Stacktrace: testST,
			})),
			expAs: Error{
				Err: fmt.Errorf("bar: %w", Error{
					Err:        errFoo,
					Ctx:        ctxA,
					Stacktrace: testST,
				}),
				Ctx:        mctx.MergeAnnotations(ctxA, ctxB),
				Stacktrace: testST,
			},
			expIs:  errFoo,
			expStr: "bar: foo",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var in Error
			ok := errors.As(test.in, &in)

			massert.Require(t, massert.Comment(
				massert.Equal(test.expAs != nil, ok),
				"test.in:%#v ok:%v", test.in, ok,
			))

			if test.expAs == nil {
				return
			}

			expAs := test.expAs.(Error)

			inAA := mctx.EvaluateAnnotations(in.Ctx, nil)
			expAsAA := mctx.EvaluateAnnotations(expAs.Ctx, nil)
			in.Ctx = nil
			expAs.Ctx = nil

			massert.Require(t,
				massert.Equal(expAsAA, inAA),
				massert.Equal(expAs, in),
				massert.Comment(
					massert.Equal(true, errors.Is(test.in, test.expIs)),
					"errors.Is(\ntest.in:%#v,\ntest.expIs:%#v,\n)", test.in, test.expIs,
				),
				massert.Equal(test.expStr, test.in.Error()),
			)
		})
	}
}
