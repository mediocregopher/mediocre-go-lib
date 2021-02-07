package mctx

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

type testAnnotator [2]string

func (t testAnnotator) Annotate(aa Annotations) {
	aa[t[0]] = t[1]
}

func TestAnnotate(t *T) {
	ctx := context.Background()
	ctx = Annotate(ctx, "a", "foo")
	ctx = Annotate(ctx, "b", "bar")
	ctx = WithAnnotator(ctx, testAnnotator{"b", "BAR"})

	aa := Annotations{}
	EvaluateAnnotations(ctx, aa)

	massert.Require(t,
		massert.Equal(Annotations{
			"a": "foo",
			"b": "BAR",
		}, aa),
	)
}

func TestAnnotationsStringMap(t *T) {
	type A int
	type B int
	aa := Annotations{
		0:    "zero",
		1:    "one",
		A(2): "two",
		B(2): "TWO",
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
	aa := Annotations{}
	EvaluateAnnotations(ctx, aa)

	err := massert.Equal(map[string]string{
		"0": "ZERO",
		"1": "ONE",
		"2": "TWO",
	}, aa.StringMap()).Assert()
	if err != nil {
		t.Fatal(err)
	}
}
