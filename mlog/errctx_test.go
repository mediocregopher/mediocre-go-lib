package mlog

import (
	"context"
	"strings"
	. "testing"

	"github.com/ansel1/merry"
	"github.com/stretchr/testify/assert"
)

func TestErrKV(t *T) {
	assertErrKV := func(err error, exp KV) {
		got := KV(ErrKV(err).KV())
		errSrc := got["errSrc"]
		assert.NotEmpty(t, errSrc)
		assert.True(t, strings.HasPrefix(errSrc.(string), "mlog/errctx_test.go"))
		delete(got, "errSrc")
		assert.Equal(t, exp, got)
	}

	err := merry.New("foo")
	assertErrKV(err, KV{"err": err.Error()})

	kv := KV{"a": "a"}
	err2 := ErrWithKV(err, kv)
	assertErrKV(err, KV{"err": err.Error()})
	assertErrKV(err2, KV{"err": err.Error(), "a": "a"})

	// changing the kv now shouldn't do anything
	kv["a"] = "b"
	assertErrKV(err, KV{"err": err.Error()})
	assertErrKV(err2, KV{"err": err.Error(), "a": "a"})

	// a new ErrWithKV shouldn't affect the previous one
	err3 := ErrWithKV(err2, KV{"b": "b"})
	assertErrKV(err, KV{"err": err.Error()})
	assertErrKV(err2, KV{"err": err2.Error(), "a": "a"})
	assertErrKV(err3, KV{"err": err3.Error(), "a": "a", "b": "b"})

	// make sure precedence works
	err4 := ErrWithKV(err3, KV{"b": "bb"})
	assertErrKV(err, KV{"err": err.Error()})
	assertErrKV(err2, KV{"err": err2.Error(), "a": "a"})
	assertErrKV(err3, KV{"err": err3.Error(), "a": "a", "b": "b"})
	assertErrKV(err4, KV{"err": err4.Error(), "a": "a", "b": "bb"})
}

func TestCtxKV(t *T) {
	ctx := context.Background()
	assert.Equal(t, KV{}, CtxKV(ctx))

	kv := KV{"a": "a"}
	ctx2 := CtxWithKV(ctx, kv)
	assert.Equal(t, KV{}, CtxKV(ctx))
	assert.Equal(t, KV{"a": "a"}, CtxKV(ctx2))

	// changing the kv now shouldn't do anything
	kv["a"] = "b"
	assert.Equal(t, KV{}, CtxKV(ctx))
	assert.Equal(t, KV{"a": "a"}, CtxKV(ctx2))

	// a new CtxWithKV shouldn't affect the previous one
	ctx3 := CtxWithKV(ctx2, KV{"b": "b"})
	assert.Equal(t, KV{}, CtxKV(ctx))
	assert.Equal(t, KV{"a": "a"}, CtxKV(ctx2))
	assert.Equal(t, KV{"a": "a", "b": "b"}, CtxKV(ctx3))

	// make sure precedence works
	ctx4 := CtxWithKV(ctx3, KV{"b": "bb"})
	assert.Equal(t, KV{}, CtxKV(ctx))
	assert.Equal(t, KV{"a": "a"}, CtxKV(ctx2))
	assert.Equal(t, KV{"a": "a", "b": "b"}, CtxKV(ctx3))
	assert.Equal(t, KV{"a": "a", "b": "bb"}, CtxKV(ctx4))
}
