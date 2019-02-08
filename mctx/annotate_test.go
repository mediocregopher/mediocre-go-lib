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
	child = Annotate(child, "a", "FOO")
	child = Annotate(child, "c", "BAZ")
	parent = WithChild(parent, child)

	parentAnnotations := LocalAnnotations(parent)
	childAnnotations := LocalAnnotations(child)
	massert.Fatal(t, massert.All(
		massert.Len(parentAnnotations, 2),
		massert.Has(parentAnnotations, [2]interface{}{"a", "foo"}),
		massert.Has(parentAnnotations, [2]interface{}{"b", "bar"}),
		massert.Len(childAnnotations, 2),
		massert.Has(childAnnotations, [2]interface{}{"a", "FOO"}),
		massert.Has(childAnnotations, [2]interface{}{"c", "BAZ"}),
	))
}

func TestAnnotationsStingMap(t *T) {
	type A int
	type B int
	aa := Annotations{
		{"foo", "bar"},
		{"1", "one"},
		{1, 1},
		{0, 0},
		{A(0), 0},
		{B(0), 0},
	}

	err := massert.Equal(map[string]string{
		"foo":       "bar",
		"string(1)": "one",
		"int(1)":    "1",
		"int(0)":    "0",
		"mctx.A(0)": "0",
		"mctx.B(0)": "0",
	}, aa.StringMap()).Assert()
	if err != nil {
		t.Fatal(err)
	}
}
