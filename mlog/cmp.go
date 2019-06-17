package mlog

import (
	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
)

type cmpKey int

// SetLogger sets the given logger onto the Component. The logger can later be
// retrieved from the Component, or any of its children, using From.
func SetLogger(cmp *mcmp.Component, l *Logger) {
	cmp.SetValue(cmpKey(0), l)
}

// DefaultLogger is an instance of Logger which is returned by From when a
// Logger hasn't been previously set with SetLogger on the passed in Component.
var DefaultLogger = NewLogger()

// From returns the Logger which was set on the Component, or one of its
// ancestors, using SetLogger. If no Logger was ever set then DefaultLogger is
// returned.
//
// The returned Logger will be modified such that it will implicitly merge the
// Contexts of any Message into the given Component's Context.
func From(cmp *mcmp.Component) *Logger {
	var l *Logger
	if l, _ = cmp.Value(cmpKey(1)).(*Logger); l != nil {
		return l
	} else if lInt, ok := cmp.InheritedValue(cmpKey(0)); ok {
		l = lInt.(*Logger)
	} else {
		l = DefaultLogger
	}

	// if we're here it means a modified Logger wasn't set on this particular
	// Component, and therefore the current one must be modified.
	l = l.Clone()
	oldHandler := l.Handler()
	l.SetHandler(func(msg Message) error {
		ctx := mctx.MergeAnnotationsInto(cmp.Context(), msg.Contexts...)
		msg.Contexts = append(msg.Contexts[:0], ctx)
		return oldHandler(msg)
	})
	cmp.SetValue(cmpKey(1), l)

	return l
}
