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

	// If the base Logger on this Component gets changed, then the cached Logger
	// from From on this Component, and all of its Children, ought to be reset,
	// so that any changes can be reflected in their loggers.
	var resetFromLogger func(*mcmp.Component)
	resetFromLogger = func(cmp *mcmp.Component) {
		cmp.SetValue(cmpKey(1), nil)
		for _, childCmp := range cmp.Children() {
			resetFromLogger(childCmp)
		}
	}
	resetFromLogger(cmp)
}

// DefaultLogger is an instance of Logger which is returned by From when a
// Logger hasn't been previously set with SetLogger on the passed in Component.
var DefaultLogger = NewLogger()

// GetLogger returns the Logger which was set on the Component, or on of its
// ancestors, using SetLogger. If no Logger was ever set then DefaultLogger is
// returned.
func GetLogger(cmp *mcmp.Component) *Logger {
	if l, ok := cmp.InheritedValue(cmpKey(0)); ok {
		return l.(*Logger)
	}
	return DefaultLogger
}

// From returns the result from GetLogger, modified so as to automatically add
// some annotations related to the Component itself to all Messages being
// logged.
func From(cmp *mcmp.Component) *Logger {
	if l, _ := cmp.Value(cmpKey(1)).(*Logger); l != nil {
		return l
	}

	// if we're here it means a modified Logger wasn't set on this particular
	// Component, and therefore the current one must be modified.
	l := GetLogger(cmp).Clone()
	oldHandler := l.Handler()
	l.SetHandler(func(msg Message) error {
		ctx := mctx.MergeAnnotationsInto(cmp.Context(), msg.Contexts...)
		msg.Contexts = append(msg.Contexts[:0], ctx)
		return oldHandler(msg)
	})
	cmp.SetValue(cmpKey(1), l)

	return l
}
