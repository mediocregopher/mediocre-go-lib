package m

import (
	"encoding/json"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestServiceCtx(t *T) {
	t.Run("log-level", func(t *T) {
		ctx := NewServiceCtx()

		// pull the Logger out of the ctx and set the Handler on it, so we can check
		// the log level
		var msgs []mlog.Message
		logger := mlog.From(ctx)
		logger.SetHandler(func(msg mlog.Message) error {
			msgs = append(msgs, msg)
			return nil
		})

		// create a child Context before running to ensure it the change propagates
		// correctly.
		ctxA := mctx.ChildOf(ctx, "A")

		params := mcfg.ParamValues{{Name: "log-level", Value: json.RawMessage(`"DEBUG"`)}}
		if err := mcfg.Populate(ctx, params); err != nil {
			t.Fatal(err)
		} else if err := mrun.Start(ctx); err != nil {
			t.Fatal(err)
		}

		mlog.From(ctxA).Info("foo")
		mlog.From(ctxA).Debug("bar")
		massert.Fatal(t, massert.All(
			massert.Len(msgs, 2),
			massert.Equal(msgs[0].Level.String(), "INFO"),
			massert.Equal(msgs[0].Description.String(), "(/A) foo"),
			massert.Equal(msgs[1].Level.String(), "DEBUG"),
			massert.Equal(msgs[1].Description.String(), "(/A) bar"),
		))
	})
}
