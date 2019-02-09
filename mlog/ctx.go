package mlog

import (
	"context"
)

type ctxKey int

// WithLogger returns the Context with the Logger carried by it.
func WithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, ctxKey(0), l)
}

// DefaultLogger is an instance of Logger which is returned by From when a
// Logger hasn't been previously WithLogger on the Contexts passed in.
var DefaultLogger = NewLogger()

// From looks at each context and returns the Logger from the first Context
// which carries one via a WithLogger call. If none carry a Logger than
// DefaultLogger is returned.
func From(ctxs ...context.Context) *Logger {
	for _, ctx := range ctxs {
		if l, _ := ctx.Value(ctxKey(0)).(*Logger); l != nil {
			return l
		}
	}
	return DefaultLogger
}

// Debug is a shortcut for
//	mlog.From(ctxs...).Debug(desc, ctxs...)
func Debug(descr string, ctxs ...context.Context) {
	From(ctxs...).Debug(descr, ctxs...)
}

// Info is a shortcut for
//	mlog.From(ctxs...).Info(desc, ctxs...)
func Info(descr string, ctxs ...context.Context) {
	From(ctxs...).Info(descr, ctxs...)
}

// Warn is a shortcut for
//	mlog.From(ctxs...).Warn(desc, ctxs...)
func Warn(descr string, ctxs ...context.Context) {
	From(ctxs...).Warn(descr, ctxs...)
}

// Error is a shortcut for
//	mlog.From(ctxs...).Error(desc, ctxs...)
func Error(descr string, ctxs ...context.Context) {
	From(ctxs...).Error(descr, ctxs...)
}

// Fatal is a shortcut for
//	mlog.From(ctxs...).Fatal(desc, ctxs...)
func Fatal(descr string, ctxs ...context.Context) {
	From(ctxs...).Fatal(descr, ctxs...)
}
