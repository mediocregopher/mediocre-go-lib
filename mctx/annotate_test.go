package mctx

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestAnnotate(t *T) {
	ctx := context.Background()
	ctx = Annotate(ctx, "a", "foo")
	ctx = Annotate(ctx, "b", "bar")
	ctx = Annotate(ctx, "b", "BAR")

	annotations := Annotations(ctx)
	massert.Require(t,
		massert.Length(annotations, 2),
		massert.HasValue(annotations, Annotation{Key: "a", Value: "foo"}),
		massert.HasValue(annotations, Annotation{Key: "b", Value: "BAR"}),
	)
}

func TestAnnotationsStringMap(t *T) {
	type A int
	type B int
	aa := AnnotationSet{
		{Key: 0, Value: "zero"},
		{Key: 1, Value: "one"},
		{Key: A(2), Value: "two"},
		{Key: B(2), Value: "TWO"},
	}

	massert.Require(t,
		massert.Equal(map[string]string{
			"0":         "zero",
			"1":         "one",
			"mctx.A(2)": "two",
			"mctx.B(2)": "TWO",
		}, aa.StringMap()),
	)
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
