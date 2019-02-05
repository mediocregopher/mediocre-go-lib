package mlog

import (
	"bytes"
	"context"
	"strings"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestContextLogging(t *T) {

	var lines []string
	l := NewLogger()
	l.SetHandler(func(msg Message) error {
		buf := new(bytes.Buffer)
		if err := DefaultFormat(buf, msg); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, strings.TrimSuffix(buf.String(), "\n"))
		return nil
	})

	ctx := Set(context.Background(), l)
	ctx1 := mctx.NewChild(ctx, "1")
	ctx1a := mctx.NewChild(ctx1, "a")
	ctx1b := mctx.NewChild(ctx1, "b")
	ctx1 = mctx.WithChild(ctx1, ctx1a)
	ctx1 = mctx.WithChild(ctx1, ctx1b)
	ctx = mctx.WithChild(ctx, ctx1)

	From(ctx).Info(ctx1a, "ctx1a")
	From(ctx).Info(ctx1, "ctx1")
	From(ctx).Info(ctx, "ctx")
	From(ctx).Debug(ctx1b, "ctx1b (shouldn't show up)")
	From(ctx).Info(ctx1b, "ctx1b")

	ctx2 := mctx.NewChild(ctx, "2")
	ctx = mctx.WithChild(ctx, ctx2)
	From(ctx2).Info(ctx2, "ctx2")

	massert.Fatal(t, massert.All(
		massert.Len(lines, 5),
		massert.Equal(lines[0], "~ INFO -- (/1/a) ctx1a"),
		massert.Equal(lines[1], "~ INFO -- (/1) ctx1"),
		massert.Equal(lines[2], "~ INFO -- ctx"),
		massert.Equal(lines[3], "~ INFO -- (/1/b) ctx1b"),
		massert.Equal(lines[4], "~ INFO -- (/2) ctx2"),
	))

	// changing MaxLevel on ctx's Logger should change it for all
	From(ctx).SetMaxLevel(DebugLevel)

	lines = lines[:0]
	From(ctx).Info(ctx, "ctx")
	From(ctx).Debug(ctx, "ctx debug")
	From(ctx2).Debug(ctx2, "ctx2 debug")

	massert.Fatal(t, massert.All(
		massert.Len(lines, 3),
		massert.Equal(lines[0], "~ INFO -- ctx"),
		massert.Equal(lines[1], "~ DEBUG -- ctx debug"),
		massert.Equal(lines[2], "~ DEBUG -- (/2) ctx2 debug"),
	))
}
