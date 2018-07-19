package mlog

import (
	"context"
	"strings"
	. "testing"

	"github.com/ansel1/merry"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestErrKV(t *T) {
	assertErrKV := func(err error, exp KV) massert.Assertion {
		got := KV(ErrKV(err).KV())
		errSrc := got["errSrc"]
		delete(got, "errSrc")
		return massert.All(
			massert.Not(massert.Nil(errSrc)),
			massert.Equal(true, strings.HasPrefix(errSrc.(string), "mlog/errctx_test.go")),
			massert.Equal(exp, got),
		)
	}

	err := merry.New("foo")
	massert.Fatal(t, assertErrKV(err, KV{"err": err.Error()}))

	kv := KV{"a": "a"}
	err2 := ErrWithKV(err, kv)
	massert.Fatal(t, massert.All(
		assertErrKV(err, KV{"err": err.Error()}),
		assertErrKV(err2, KV{"err": err.Error(), "a": "a"}),
	))

	// changing the kv now shouldn't do anything
	kv["a"] = "b"
	massert.Fatal(t, massert.All(
		assertErrKV(err, KV{"err": err.Error()}),
		assertErrKV(err2, KV{"err": err.Error(), "a": "a"}),
	))

	// a new ErrWithKV shouldn't affect the previous one
	err3 := ErrWithKV(err2, KV{"b": "b"})
	massert.Fatal(t, massert.All(
		assertErrKV(err, KV{"err": err.Error()}),
		assertErrKV(err2, KV{"err": err2.Error(), "a": "a"}),
		assertErrKV(err3, KV{"err": err3.Error(), "a": "a", "b": "b"}),
	))

	// make sure precedence works
	err4 := ErrWithKV(err3, KV{"b": "bb"})
	massert.Fatal(t, massert.All(
		assertErrKV(err, KV{"err": err.Error()}),
		assertErrKV(err2, KV{"err": err2.Error(), "a": "a"}),
		assertErrKV(err3, KV{"err": err3.Error(), "a": "a", "b": "b"}),
		assertErrKV(err4, KV{"err": err4.Error(), "a": "a", "b": "bb"}),
	))
}

func TestCtxKV(t *T) {
	ctx := context.Background()
	massert.Fatal(t, massert.Equal(KV{}, CtxKV(ctx)))

	kv := KV{"a": "a"}
	ctx2 := CtxWithKV(ctx, kv)
	massert.Fatal(t, massert.All(
		massert.Equal(KV{}, CtxKV(ctx)),
		massert.Equal(KV{"a": "a"}, CtxKV(ctx2)),
	))

	// changing the kv now shouldn't do anything
	kv["a"] = "b"
	massert.Fatal(t, massert.All(
		massert.Equal(KV{}, CtxKV(ctx)),
		massert.Equal(KV{"a": "a"}, CtxKV(ctx2)),
	))

	// a new CtxWithKV shouldn't affect the previous one
	ctx3 := CtxWithKV(ctx2, KV{"b": "b"})
	massert.Fatal(t, massert.All(
		massert.Equal(KV{}, CtxKV(ctx)),
		massert.Equal(KV{"a": "a"}, CtxKV(ctx2)),
		massert.Equal(KV{"a": "a", "b": "b"}, CtxKV(ctx3)),
	))

	// make sure precedence works
	ctx4 := CtxWithKV(ctx3, KV{"b": "bb"})
	massert.Fatal(t, massert.All(
		massert.Equal(KV{}, CtxKV(ctx)),
		massert.Equal(KV{"a": "a"}, CtxKV(ctx2)),
		massert.Equal(KV{"a": "a", "b": "b"}, CtxKV(ctx3)),
		massert.Equal(KV{"a": "a", "b": "bb"}, CtxKV(ctx4)),
	))
}
