package mlog

import (
	"fmt"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestGetSetLogger(t *T) {
	cmp := new(mcmp.Component)
	cmpChild := cmp.Child("child")
	ctx := mctx.Annotated("foo", "bar")

	var msgs []string
	l := NewLogger()
	l.SetHandler(func(msg Message) error {
		msgStr := fmt.Sprintf("%s %q", msg.Level, msg.Description)
		for _, ctx := range msg.Contexts {
			for _, kv := range mctx.Annotations(ctx).StringSlice(true) {
				msgStr += fmt.Sprintf(" %s=%s", kv[0], kv[1])
			}
		}
		msgs = append(msgs, msgStr)
		return nil
	})
	SetLogger(cmp, l)

	msgs = msgs[:0]
	GetLogger(cmp).Info("get-cmp", ctx)
	GetLogger(cmpChild).Info("get-cmpChild", ctx)
	From(cmp).Info("from-cmp", ctx)
	From(cmpChild).Info("from-cmpChild", ctx)
	massert.Require(t,
		massert.Equal(`INFO "get-cmp" foo=bar`, msgs[0]),
		massert.Equal(`INFO "get-cmpChild" foo=bar`, msgs[1]),
		massert.Equal(`INFO "from-cmp" componentPath=/ foo=bar`, msgs[2]),
		massert.Equal(`INFO "from-cmpChild" componentPath=/child foo=bar`, msgs[3]),
	)

	l2 := l.Clone()
	l2.SetHandler(func(msg Message) error {
		msg.Description += " (2)"
		return l.Handler()(msg)
	})
	SetLogger(cmp, l2)

	msgs = msgs[:0]
	GetLogger(cmp).Info("get-cmp", ctx)
	GetLogger(cmpChild).Info("get-cmpChild", ctx)
	From(cmp).Info("from-cmp", ctx)
	From(cmpChild).Info("from-cmpChild", ctx)
	massert.Require(t,
		massert.Equal(`INFO "get-cmp (2)" foo=bar`, msgs[0]),
		massert.Equal(`INFO "get-cmpChild (2)" foo=bar`, msgs[1]),
		massert.Equal(`INFO "from-cmp (2)" componentPath=/ foo=bar`, msgs[2]),
		massert.Equal(`INFO "from-cmpChild (2)" componentPath=/child foo=bar`, msgs[3]),
	)

	// If a Logger is set on the child, that shouldn't affect the parent
	l3 := l.Clone()
	l3.SetHandler(func(msg Message) error {
		msg.Description += " (3)"
		return l.Handler()(msg)
	})
	SetLogger(cmpChild, l3)

	msgs = msgs[:0]
	GetLogger(cmp).Info("get-cmp", ctx)
	GetLogger(cmpChild).Info("get-cmpChild", ctx)
	From(cmp).Info("from-cmp", ctx)
	From(cmpChild).Info("from-cmpChild", ctx)
	massert.Require(t,
		massert.Equal(`INFO "get-cmp (2)" foo=bar`, msgs[0]),
		massert.Equal(`INFO "get-cmpChild (3)" foo=bar`, msgs[1]),
		massert.Equal(`INFO "from-cmp (2)" componentPath=/ foo=bar`, msgs[2]),
		massert.Equal(`INFO "from-cmpChild (3)" componentPath=/child foo=bar`, msgs[3]),
	)

}
