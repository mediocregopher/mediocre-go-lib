package mlog

import "context"

type ctxKey int

// Set returns the Context with the Logger carried by it.
func Set(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, ctxKey(0), l)
}

// DefaultLogger is an instance of Logger which is returned by From when a
// Logger hasn't been previously Set on the Context passed in.
var DefaultLogger = NewLogger()

// From returns the Logger carried by this Context, or DefaultLogger if none is
// being carried.
func From(ctx context.Context) *Logger {
	if l, _ := ctx.Value(ctxKey(0)).(*Logger); l != nil {
		return l
	}
	return DefaultLogger
}

// Debug is a shortcut for
//	mlog.From(ctx).Debug(ctx, descr, kvs...)
func Debug(ctx context.Context, descr string, kvs ...KVer) {
	From(ctx).Debug(ctx, descr, kvs...)
}

// Info is a shortcut for
//	mlog.From(ctx).Info(ctx, descr, kvs...)
func Info(ctx context.Context, descr string, kvs ...KVer) {
	From(ctx).Info(ctx, descr, kvs...)
}

// Warn is a shortcut for
//	mlog.From(ctx).Warn(ctx, descr, kvs...)
func Warn(ctx context.Context, descr string, kvs ...KVer) {
	From(ctx).Warn(ctx, descr, kvs...)
}

// Error is a shortcut for
//	mlog.From(ctx).Error(ctx, descr, kvs...)
func Error(ctx context.Context, descr string, kvs ...KVer) {
	From(ctx).Error(ctx, descr, kvs...)
}

// Fatal is a shortcut for
//	mlog.From(ctx).Fatal(ctx, descr, kvs...)
func Fatal(ctx context.Context, descr string, kvs ...KVer) {
	From(ctx).Fatal(ctx, descr, kvs...)
}
