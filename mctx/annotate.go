package mctx

import (
	"context"
	"fmt"
	"sort"
)

// Annotation describes the annotation of a key/value pair made on a Context via
// the Annotate call.
type Annotation struct {
	Key, Value interface{}
}

type annotation struct {
	Annotation
	root, prev *annotation
}

type annotationKey int

// Annotate takes in one or more key/value pairs (kvs' length must be even) and
// returns a Context carrying them.
func Annotate(ctx context.Context, kvs ...interface{}) context.Context {
	if len(kvs)%2 > 0 {
		panic("kvs being passed to mctx.Annotate must have an even number of elements")
	} else if len(kvs) == 0 {
		return ctx
	}

	// if multiple annotations are passed in here it's not actually necessary to
	// create an intermediate Context for each one, so keep curr outside and
	// only use it later
	var curr, root *annotation
	prev, _ := ctx.Value(annotationKey(0)).(*annotation)
	if prev != nil {
		root = prev.root
	}
	for i := 0; i < len(kvs); i += 2 {
		curr = &annotation{
			Annotation: Annotation{Key: kvs[i], Value: kvs[i+1]},
			prev:       prev,
		}
		if root == nil {
			root = curr
		}
		curr.root = curr
		prev = curr
	}

	ctx = context.WithValue(ctx, annotationKey(0), curr)
	return ctx
}

// Annotated is a shortcut for calling Annotate with a context.Background().
func Annotated(kvs ...interface{}) context.Context {
	return Annotate(context.Background(), kvs...)
}

// AnnotationSet describes a set of unique Annotation values which were
// retrieved off a Context via the Annotations function. An AnnotationSet has a
// couple methods on it to aid in post-processing.
type AnnotationSet []Annotation

// Annotations returns all Annotation values which have been set via Annotate on
// this Context and its ancestors. If a key was set twice then only the most
// recent value is included.
func Annotations(ctx context.Context) AnnotationSet {
	a, _ := ctx.Value(annotationKey(0)).(*annotation)
	if a == nil {
		return nil
	}
	m := map[interface{}]bool{}

	var aa AnnotationSet
	for {
		if a == nil {
			break
		}

		if m[a.Key] {
			a = a.prev
			continue
		}

		aa = append(aa, a.Annotation)
		m[a.Key] = true
		a = a.prev
	}
	return aa
}

// StringMap formats each of the Annotations into strings using fmt.Sprint. If
// any two keys format to the same string, then type information will be
// prefaced to each one.
func (aa AnnotationSet) StringMap() map[string]string {
	type mKey struct {
		str string
		typ string
	}
	m := map[mKey][]Annotation{}
	for _, a := range aa {
		k := mKey{str: fmt.Sprint(a.Key)}
		m[k] = append(m[k], a)
	}

	nextK := func(k mKey, a Annotation) mKey {
		if k.typ == "" {
			k.typ = fmt.Sprintf("%T", a.Key)
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
			for _, a := range annotations {
				k2 := nextK(k, a)
				m[k2] = append(m[k2], a)
			}
			delete(m, k)
		}
		if !any {
			break
		}
	}

	outM := map[string]string{}
	for k, annotations := range m {
		a := annotations[0]
		kStr := k.str
		if k.typ != "" {
			kStr = k.typ + "(" + kStr + ")"
		}
		outM[kStr] = fmt.Sprint(a.Value)
	}
	return outM
}

// StringSlice is like StringMap but it returns a slice of key/value tuples
// rather than a map. If sorted is true then the slice will be sorted by key in
// ascending order.
func (aa AnnotationSet) StringSlice(sorted bool) [][2]string {
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

func mergeAnnotations(ctxA, ctxB context.Context) context.Context {
	annotationA, _ := ctxA.Value(annotationKey(0)).(*annotation)
	annotationB, _ := ctxB.Value(annotationKey(0)).(*annotation)
	if annotationB == nil {
		return ctxA
	} else if annotationA == nil {
		return context.WithValue(ctxA, annotationKey(0), annotationB)
	}

	var headA, currA *annotation
	currB := annotationB
	for {
		if currB == nil {
			break
		}

		prevA := &annotation{
			Annotation: currB.Annotation,
			root:       annotationA.root,
		}
		if currA != nil {
			currA.prev = prevA
		}
		currA, currB = prevA, currB.prev
		if headA == nil {
			headA = currA
		}
	}

	currA.prev = annotationA
	return context.WithValue(ctxA, annotationKey(0), headA)
}

// MergeAnnotations sequentially merges the annotation data of the passed in
// Contexts into the first passed in one. Data from a Context overwrites
// overlapping data on all passed in Contexts to the left of it. All other
// aspects of the first Context remain the same, and that Context is returned
// with the new set of Annotation data.
//
// NOTE this will panic if no Contexts are passed in.
func MergeAnnotations(ctxs ...context.Context) context.Context {
	return MergeAnnotationsInto(ctxs[0], ctxs[1:]...)
}

// MergeAnnotationsInto is a convenience function which works like
// MergeAnnotations.
func MergeAnnotationsInto(ctx context.Context, ctxs ...context.Context) context.Context {
	for _, ctxB := range ctxs {
		ctx = mergeAnnotations(ctx, ctxB)
	}
	return ctx
}
