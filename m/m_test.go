package m

import (
	"context"
	"encoding/json"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestServiceCtx(t *T) {
	t.Run("log-level", func(t *T) {
		cmp := RootComponent()

		// pull the Logger out of the component and set the Handler on it, so we
		// can check the log level
		var msgs []mlog.Message
		logger := mlog.GetLogger(cmp)
		logger.SetHandler(func(msg mlog.Message) error {
			msgs = append(msgs, msg)
			return nil
		})
		mlog.SetLogger(cmp, logger)

		// create a child Component before running to ensure it the change propagates
		// correctly.
		cmpA := cmp.Child("A")

		params := mcfg.ParamValues{{Name: "log-level", Value: json.RawMessage(`"DEBUG"`)}}
		cmp.SetValue(cmpKeyCfgSrc, params)
		MustInit(cmp)

		mlog.From(cmpA).Info("foo")
		mlog.From(cmpA).Debug("bar")
		massert.Require(t,
			massert.Length(msgs, 2),
			massert.Equal(msgs[0].Level.String(), "INFO"),
			massert.Equal(msgs[0].Description, "foo"),
			massert.Equal(msgs[0].Contexts, []context.Context{cmpA.Context()}),
			massert.Equal(msgs[1].Level.String(), "DEBUG"),
			massert.Equal(msgs[1].Description, "bar"),
			massert.Equal(msgs[1].Contexts, []context.Context{cmpA.Context()}),
		)
	})
}
