package mctx

import (
	"context"
	"fmt"
	"sort"
)

type ctxKeyAnnotation int

// Annotator is a type which can add annotation data to an existing set of
// Annotations. The Annotate method should be expected to be called in a
// non-thread-safe manner.
type Annotator interface {
	Annotate(Annotations)
}

type el struct {
	annotator Annotator
	prev      *el
}

// WithAnnotator takes in an Annotator and returns a Context which will produce
// that Annotator's annotations when the Annotate function is called. The
// Annotator will be not be evaluated until the first call to Annotate.
func WithAnnotator(ctx context.Context, annotator Annotator) context.Context {
	curr := &el{annotator: annotator}
	curr.prev, _ = ctx.Value(ctxKeyAnnotation(0)).(*el)
	return context.WithValue(ctx, ctxKeyAnnotation(0), curr)
}

type annotationSeq []interface{}

func (s annotationSeq) Annotate(aa Annotations) {
	for i := 0; i < len(s); i += 2 {
		aa[s[i]] = s[i+1]
	}
}

// Annotate is a shortcut for calling WithAnnotator using an Annotations
// containing the given key/value pairs.
//
// NOTE If the length of kvs is not divisible by two this will panic.
func Annotate(ctx context.Context, kvs ...interface{}) context.Context {
	if len(kvs)%2 > 0 {
		panic("kvs being passed to mctx.Annotate must have an even number of elements")
	} else if len(kvs) == 0 {
		return ctx
	}
	return WithAnnotator(ctx, annotationSeq(kvs))
}

// Annotations is a set of key/value pairs representing a set of annotations. It
// implements the Annotator interface along with other useful post-processing
// methods.
type Annotations map[interface{}]interface{}

// Annotate implements the method for the Annotator interface.
func (aa Annotations) Annotate(aa2 Annotations) {
	for k, v := range aa {
		aa2[k] = v
	}
}

// StringMap formats each of the key/value pairs into strings using fmt.Sprint.
// If any two keys format to the same string, then type information will be
// prefaced to each one.
func (aa Annotations) StringMap() map[string]string {
	type mKey struct {
		str string
		typ string
	}
	m := map[mKey][][2]interface{}{}
	for k, v := range aa {
		mk := mKey{str: fmt.Sprint(k)}
		m[mk] = append(m[mk], [2]interface{}{k, v})
	}

	nextK := func(k mKey, kv [2]interface{}) mKey {
		if k.typ == "" {
			k.typ = fmt.Sprintf("%T", kv[0])
		} else {
			panic(fmt.Sprintf("mKey %#v is somehow conflicting with another", k))
		}
		return k
	}

	for {
		var any bool
		for k, annotations := range m {
			if len(annotations) == 1 {
				continue
			}
			any = true
			for _, kv := range annotations {
				k2 := nextK(k, kv)
				m[k2] = append(m[k2], kv)
			}
			delete(m, k)
		}
		if !any {
			break
		}
	}

	outM := map[string]string{}
	for k, annotations := range m {
		kv := annotations[0]
		kStr := k.str
		if k.typ != "" {
			kStr = k.typ + "(" + kStr + ")"
		}
		outM[kStr] = fmt.Sprint(kv[1])
	}
	return outM
}

// StringSlice is like StringMap but it returns a slice of key/value tuples
// rather than a map. If sorted is true then the slice will be sorted by key in
// ascending order.
func (aa Annotations) StringSlice(sorted bool) [][2]string {
	m := aa.StringMap()
	slice := make([][2]string, 0, len(m))
	for k, v := range m {
		slice = append(slice, [2]string{k, v})
	}
	if sorted {
		sort.Slice(slice, func(i, j int) bool {
			return slice[i][0] < slice[j][0]
		})
	}
	return slice
}

// EvaluateAnnotations collects all annotation key/values which have been set
// via Annotate(With) on this Context and its ancestors, and sets those
// key/values on the given Annotations. If a key was set twice then only the
// most recent value is included.
//
// For convenience the passed in Annotations is returned from this function, and
// if nil is given as the Annotations value then an Annotations will be
// allocated and returned.
func EvaluateAnnotations(ctx context.Context, aa Annotations) Annotations {
	if aa == nil {
		aa = Annotations{}
	}

	tmp := Annotations{}
	for el, _ := ctx.Value(ctxKeyAnnotation(0)).(*el); el != nil; el = el.prev {
		el.annotator.Annotate(tmp)
		for k, v := range tmp {
			if _, ok := aa[k]; ok {
				continue
			}
			aa[k] = v
			delete(tmp, k)
		}
	}
	return aa
}

//
// MergeAnnotations sequentially merges the annotation data of the passed in
// Contexts into the first passed in Context. Data from a Context overwrites
// overlapping data on all passed in Contexts to the left of it. All other
// aspects of the first Context remain the same, and that Context is returned
// with the new set of Annotation data.
func MergeAnnotations(ctx context.Context, ctxs ...context.Context) context.Context {
	aa := Annotations{}
	tmp := Annotations{}
	EvaluateAnnotations(ctx, aa)
	for _, ctxB := range ctxs {
		EvaluateAnnotations(ctxB, tmp)
		for k, v := range tmp {
			aa[k] = v
			delete(tmp, k)
		}
	}
	return context.WithValue(ctx, ctxKeyAnnotation(0), &el{annotator: aa})
}
