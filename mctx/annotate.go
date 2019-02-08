package mctx

import (
	"context"
	"fmt"
)

type annotateKey struct {
	userKey interface{}
}

// Annotate takes in one or more key/value pairs (kvs' length must be even) and
// returns a Context carrying them. Annotations only exist on the local level,
// i.e. a child and parent share different annotation namespaces.
func Annotate(ctx context.Context, kvs ...interface{}) context.Context {
	for i := 0; i < len(kvs); i += 2 {
		ctx = WithLocalValue(ctx, annotateKey{kvs[i]}, kvs[i+1])
	}
	return ctx
}

// Annotations describes a set of keys/values which were set on a Context (but
// not its parents or children) using Annotate.
type Annotations [][2]interface{}

// LocalAnnotations returns all key/value pairs which have been set via Annotate
// on this Context (but not its parent or children). If a key was set twice then
// only the most recent value is included. The returned order is
// non-deterministic.
func LocalAnnotations(ctx context.Context) Annotations {
	var annotations Annotations
	localValuesIter(ctx, func(key, val interface{}) {
		aKey, ok := key.(annotateKey)
		if !ok {
			return
		}
		annotations = append(annotations, [2]interface{}{aKey.userKey, val})
	})
	return annotations
}

// StringMap formats each of the key/value pairs into strings using fmt.Sprint.
// If any two keys format to the same string, then type information will be
// prefaced to each one.
func (aa Annotations) StringMap() map[string]string {
	keyTypes := make(map[string]interface{}, len(aa))
	keyVals := map[string]string{}
	setKey := func(k, v interface{}) {
		kStr := fmt.Sprint(k)
		oldType := keyTypes[kStr]
		if oldType == nil {
			keyTypes[kStr] = k
			keyVals[kStr] = fmt.Sprint(v)
			return
		}

		// check if oldKey is in kvm, if so it needs to be moved to account for
		// its type info
		if oldV, ok := keyVals[kStr]; ok {
			delete(keyVals, kStr)
			keyVals[fmt.Sprintf("%T(%s)", oldType, kStr)] = oldV
		}

		keyVals[fmt.Sprintf("%T(%s)", k, kStr)] = fmt.Sprint(v)
	}

	for _, kv := range aa {
		setKey(kv[0], kv[1])
	}
	return keyVals
}
