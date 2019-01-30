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

// CtxSetAll traverses the given Context's children, breadth-first. It calls the
// callback for each Context which has a Logger set on it, replacing that Logger
// with the returned one.
//
// This is useful, for example, when changing the log level of all Loggers in an
// app.
func CtxSetAll(ctx mctx.Context, callback func(mctx.Context, *Logger) *Logger) {
	mctx.BreadthFirstVisit(ctx, func(ctx mctx.Context) bool {
		mctx.GetSetMutableValue(ctx, false, ctxKey(0), func(i interface{}) interface{} {
			if i == nil {
				return nil
			}
			return callback(ctx, i.(*Logger))
		})
		return true
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
	return mctx.GetSetMutableValue(ctx, true, ctxKey(0), func(interface{}) interface{} {
		ctxPath := mctx.Path(ctx)
		if len(ctxPath) == 0 {
			// we're at the root node and it doesn't have a Logger set, use
			// the default
			return NewLogger()
		}

		// set up child's logger
		pathStr := "/" + path.Join(ctxPath...)

		parentL := From(mctx.Parent(ctx))
		parentH := parentL.Handler()
		thisL := parentL.Clone()
		thisL.SetHandler(func(msg Message) error {
			// if the Description is already a ctxPathStringer it can be
			// assumed this Message was passed in from a child Logger.
			if _, ok := msg.Description.(ctxPathStringer); !ok {
				msg.Description = ctxPathStringer{
					str:     msg.Description,
					pathStr: pathStr,
				}
			}
			return parentH(msg)
		})
		return thisL
	}).(*Logger)
}
