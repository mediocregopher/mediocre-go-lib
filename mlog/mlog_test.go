package mlog

import (
	"bytes"
	"context"
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
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
	h := defaultHandler(buf)

	l := NewLogger()
	l.SetHandler(h)
	l.testMsgWrittenCh = make(chan struct{}, 10)

	assertOut := func(expected string) massert.Assertion {
		select {
		case <-l.testMsgWrittenCh:
		case <-time.After(1 * time.Second):
			return massert.Errorf("waited too long for msg to write")
		}
		out, err := buf.ReadString('\n')
		return massert.All(
			massert.Nil(err),
			massert.Equal(expected, strings.TrimSpace(out)),
		)
	}

	// Default max level should be INFO
	l.Debug("foo")
	l.Info("bar")
	l.Warn("baz")
	l.Error("buz")
	massert.Require(t,
		assertOut(`{"level":"INFO","descr":"bar"}`),
		assertOut(`{"level":"WARN","descr":"baz"}`),
		assertOut(`{"level":"ERROR","descr":"buz"}`),
	)

	ctx := context.Background()

	l.SetMaxLevel(WarnLevel)
	l.Debug("foo")
	l.Info("bar")
	l.Warn("baz")
	l.Error("buz", mctx.Annotate(ctx, "a", "b", "c", "d"))
	massert.Require(t,
		assertOut(`{"level":"WARN","descr":"baz"}`),
		assertOut(`{"level":"ERROR","descr":"buz","annotations":{"a":"b","c":"d"}}`),
	)

	l2 := l.Clone()
	l2.SetMaxLevel(InfoLevel)
	l2.SetHandler(func(msg Message) error {
		msg.Description = strings.ToUpper(msg.Description)
		return h(msg)
	})
	l2.Info("bar")
	l2.Warn("baz")
	l.Error("buz")
	massert.Require(t,
		assertOut(`{"level":"INFO","descr":"BAR"}`),
		assertOut(`{"level":"WARN","descr":"BAZ"}`),
		assertOut(`{"level":"ERROR","descr":"buz"}`),
	)
}
