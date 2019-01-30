package mlog

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestContextStuff(t *T) {
	ctx := mctx.New()
	ctx1 := mctx.ChildOf(ctx, "1")
	ctx1a := mctx.ChildOf(ctx1, "a")
	ctx1b := mctx.ChildOf(ctx1, "b")

	var descrs []string
	l := NewLogger()
	l.SetHandler(func(msg Message) error {
		descrs = append(descrs, msg.Description.String())
		return nil
	})
	CtxSet(ctx, l)

	From(ctx1a).Info("ctx1a")
	From(ctx1).Info("ctx1")
	From(ctx).Info("ctx")
	From(ctx1b).Debug("ctx1b (shouldn't show up)")
	From(ctx1b).Info("ctx1b")

	ctx2 := mctx.ChildOf(ctx, "2")
	From(ctx2).Info("ctx2")

	massert.Fatal(t, massert.All(
		massert.Len(descrs, 5),
		massert.Equal(descrs[0], "(/1/a) ctx1a"),
		massert.Equal(descrs[1], "(/1) ctx1"),
		massert.Equal(descrs[2], "ctx"),
		massert.Equal(descrs[3], "(/1/b) ctx1b"),
		massert.Equal(descrs[4], "(/2) ctx2"),
	))

	// use CtxSetAll to change all MaxLevels in-place
	ctx2L := From(ctx2)
	CtxSetAll(ctx, func(_ mctx.Context, l *Logger) *Logger {
		l.SetMaxLevel(DebugLevel)
		return l
	})

	descrs = descrs[:0]
	From(ctx).Info("ctx")
	From(ctx).Debug("ctx debug")
	ctx2L.Debug("ctx2L debug")

	massert.Fatal(t, massert.All(
		massert.Len(descrs, 3),
		massert.Equal(descrs[0], "ctx"),
		massert.Equal(descrs[1], "ctx debug"),
		massert.Equal(descrs[2], "(/2) ctx2L debug"),
	))
}
