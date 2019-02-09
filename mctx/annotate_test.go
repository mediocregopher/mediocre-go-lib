package mctx

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestAnnotate(t *T) {
	parent := context.Background()
	parent = Annotate(parent, "a", "foo")
	parent = Annotate(parent, "b", "bar")

	child := NewChild(parent, "child")
	child = Annotate(child, "a", "Foo")
	child = Annotate(child, "a", "FOO")
	child = Annotate(child, "c", "BAZ")
	parent = WithChild(parent, child)

	parentAnnotations := Annotations(parent)
	childAnnotations := Annotations(child)
	massert.Fatal(t, massert.All(
		massert.Len(parentAnnotations, 2),
		massert.Has(parentAnnotations, Annotation{Key: "a", Value: "foo"}),
		massert.Has(parentAnnotations, Annotation{Key: "b", Value: "bar"}),

		massert.Len(childAnnotations, 4),
		massert.Has(childAnnotations, Annotation{Key: "a", Value: "foo"}),
		massert.Has(childAnnotations, Annotation{Key: "b", Value: "bar"}),
		massert.Has(childAnnotations,
			Annotation{Key: "a", Path: []string{"child"}, Value: "FOO"}),
		massert.Has(childAnnotations,
			Annotation{Key: "c", Path: []string{"child"}, Value: "BAZ"}),
	))
}

func TestAnnotationsStringMap(t *T) {
	type A int
	type B int
	aa := AnnotationSet{
		{Key: 0, Path: nil, Value: "zero"},
		{Key: 1, Path: nil, Value: "one"},
		{Key: 1, Path: []string{"foo"}, Value: "ONE"},
		{Key: A(2), Path: []string{"foo"}, Value: "two"},
		{Key: B(2), Path: []string{"foo"}, Value: "TWO"},
	}

	massert.Fatal(t, massert.All(
		massert.Equal(map[string]string{
			"0":               "zero",
			"1(/)":            "one",
			"1(/foo)":         "ONE",
			"2(/foo)(mctx.A)": "two",
			"2(/foo)(mctx.B)": "TWO",
		}, aa.StringMap()),
		massert.Equal(map[string]map[string]string{
			"/": {
				"0": "zero",
				"1": "one",
			},
			"/foo": {
				"1":         "ONE",
				"2(mctx.A)": "two",
				"2(mctx.B)": "TWO",
			},
		}, aa.StringMapByPath()),
	))
}

func TestMergeAnnotations(t *T) {
	ctxA := Annotate(context.Background(), 0, "zero", 1, "one")
	ctxA = Annotate(ctxA, 0, "ZERO")
	ctxB := Annotate(context.Background(), 2, "two")
	ctxB = Annotate(ctxB, 1, "ONE", 2, "TWO")

	ctx := MergeAnnotations(ctxA, ctxB)
	err := massert.Equal(map[string]string{
		"0": "ZERO",
		"1": "ONE",
		"2": "TWO",
	}, Annotations(ctx).StringMap()).Assert()
	if err != nil {
		t.Fatal(err)
	}
}
