package mlog

import (
	"path"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
)

type ctxKey int

// CtxSet caches the Logger in the Context, overwriting any previous one which
// might have been cached there. From is the corresponding function which
// retrieves the Logger back out when needed.
//
// This function can be used to premptively set a preconfigured Logger on a root
// Context so that the default (NewLogger) isn't used when From is called for
// the first time.
func CtxSet(ctx mctx.Context, l *Logger) {
	mctx.GetSetMutableValue(ctx, false, ctxKey(0), func(interface{}) interface{} {
		return l
	})
}

type ctxPathStringer struct {
	str     Stringer
	pathStr string
}

func (cp ctxPathStringer) String() string {
	return "(" + cp.pathStr + ") " + cp.str.String()
}

// From returns an instance of Logger which has been customized for this
// Context, primarily by adding a prefix describing the Context's path to all
// Message descriptions the Logger receives.
//
// The Context caches within it the generated Logger, so a new one isn't created
// everytime. When From is first called on a Context the Logger inherits the
// Context parent's Logger. If the parent hasn't had From called on it its
// parent will be queried instead, and so on.
func From(ctx mctx.Context) *Logger {
	var from func(mctx.Context) *Logger
	from = func(ctx mctx.Context) *Logger {
		return mctx.GetSetMutableValue(ctx, true, ctxKey(0), func(interface{}) interface{} {

			ctxPath := mctx.Path(ctx)
			if len(ctxPath) == 0 {
				// we're at the root node and it doesn't have a Logger set, use
				// the default
				return NewLogger()
			}

			// set up child's logger
			pathStr := "/"
			if len(ctxPath) > 0 {
				pathStr += path.Join(ctxPath...)
			}

			parentL := from(mctx.Parent(ctx))
			prevH := parentL.Handler()
			return parentL.WithHandler(func(msg Message) error {
				// if the Description is already a ctxPathStringer it can be
				// assumed this Message was passed in from a child Logger.
				if _, ok := msg.Description.(ctxPathStringer); !ok {
					msg.Description = ctxPathStringer{
						str:     msg.Description,
						pathStr: pathStr,
					}
				}
				return prevH(msg)
			})

		}).(*Logger)
	}

	return from(ctx)
}
