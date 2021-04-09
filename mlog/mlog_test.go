package mlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/v2/mctx"
	"github.com/mediocregopher/mediocre-go-lib/v2/mtest/massert"
)

func TestTruncate(t *T) {
	massert.Require(t,
		massert.Equal("abc", Truncate("abc", 4)),
		massert.Equal("abc", Truncate("abc", 3)),
		massert.Equal("ab...", Truncate("abc", 2)),
	)
}

func TestLogger(t *T) {
	buf := new(bytes.Buffer)
	now := time.Now().UTC()
	td, ts := now.Format(msgTimeFormat), fmt.Sprint(now.UnixNano())

	l := NewLogger(&LoggerOpts{
		MessageHandler: NewMessageHandler(buf),
		Now:            func() time.Time { return now },
	})

	assertOut := func(expected string) massert.Assertion {
		expected = strings.ReplaceAll(expected, "<TD>", td)
		expected = strings.ReplaceAll(expected, "<TS>", ts)
		out, err := buf.ReadString('\n')
		return massert.All(
			massert.Nil(err),
			massert.Equal(expected, strings.TrimSpace(out)),
		)
	}

	ctx := context.Background()

	// Default max level should be INFO
	l.Debug(ctx, "foo")
	l.Info(ctx, "bar")
	l.Warn(ctx, "baz", errors.New("ERR"))
	l.Error(ctx, "buz", errors.New("ERR"))
	massert.Require(t,
		assertOut(`{"td":"<TD>","ts":<TS>,"level":"INFO","descr":"bar","level_int":30}`),
		assertOut(`{"td":"<TD>","ts":<TS>,"level":"WARN","descr":"baz: ERR","level_int":20}`),
		assertOut(`{"td":"<TD>","ts":<TS>,"level":"ERROR","descr":"buz: ERR","level_int":10}`),
	)

	// annotate context
	ctx = mctx.Annotate(ctx, "foo", "bar")
	l.Info(ctx, "bar")
	massert.Require(t,
		assertOut(`{"td":"<TD>","ts":<TS>,"level":"INFO","descr":"bar","level_int":30,"annotations":{"foo":"bar"}}`),
	)

	// add namespace
	l = l.WithNamespace("ns")
	l.Info(ctx, "bar")
	massert.Require(t,
		assertOut(`{"td":"<TD>","ts":<TS>,"level":"INFO","ns":["ns"],"descr":"bar","level_int":30,"annotations":{"foo":"bar"}}`),
	)
}
