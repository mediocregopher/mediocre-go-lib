package mlog

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ansel1/merry"
)

type kvKey int

// ErrWithKV embeds the merging of a set of KVs into an error, returning a new
// error instance. If the error already has a KV embedded in it then the
// returned error will have the merging of them all, with the given KVs taking
// precedence.
func ErrWithKV(err error, kvs ...KVer) merry.Error {
	if err == nil {
		return nil
	}
	merr := merry.WrapSkipping(err, 1)
	var kv KV
	if exKV := merry.Value(merr, kvKey(0)); exKV != nil {
		kv = mergeInto(exKV.(KV), kvs...)
	} else {
		kv = merge(kvs...)
	}
	return merr.WithValue(kvKey(0), kv)
}

// ErrKV returns a KV which includes the KV embedded in the error by ErrWithKV,
// if any. The KV will also include an "err" field containing the output of
// err.Error(), and an "errSrc" field if the given error is a merry error which
// contains an embedded stacktrace.
func ErrKV(err error) KVer {
	var kv KV
	if kvi := merry.Value(err, kvKey(0)); kvi != nil {
		kv = kvi.(KV)
	} else {
		kv = KV{}
	}
	kv["err"] = err.Error()
	if fileFull, line := merry.Location(err); fileFull != "" {
		file, dir := filepath.Base(fileFull), filepath.Dir(fileFull)
		dir = filepath.Base(dir) // only want the first dirname, ie the pkg name
		kv["errSrc"] = fmt.Sprintf("%s/%s:%d", dir, file, line)
	}
	return kv
}

// CtxWithKV embeds a KV into a Context, returning a new Context instance. If
// the Context already has a KV embedded in it then the returned error will have
// the merging of the two, with the given KVs taking precedence.
func CtxWithKV(ctx context.Context, kvs ...KVer) context.Context {
	existingKV := ctx.Value(kvKey(0))
	var kv KV
	if existingKV != nil {
		kv = mergeInto(existingKV.(KV), kvs...)
	} else {
		kv = merge(kvs...)
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
