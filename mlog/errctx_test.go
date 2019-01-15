package mlog

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

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
