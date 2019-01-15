package mlog

import (
	"context"
)

type kvKey int

// CtxWithKV embeds a KV into a Context, returning a new Context instance. If
// the Context already has a KV embedded in it then the returned error will have
// the merging of the two, with the given KVs taking precedence.
func CtxWithKV(ctx context.Context, kvs ...KVer) context.Context {
	existingKV := ctx.Value(kvKey(0))
	var kv KV
	if existingKV != nil {
		kv = mergeInto(existingKV.(KV), kvs...)
	} else {
		kv = Merge(kvs...).KV()
	}
	return context.WithValue(ctx, kvKey(0), kv)
}

// CtxKV returns a copy of the KV embedded in the Context by CtxWithKV
func CtxKV(ctx context.Context) KVer {
	kv := ctx.Value(kvKey(0))
	if kv == nil {
		return KV{}
	}
	return kv.(KV)
}
