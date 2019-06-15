package mlog

import (
	"github.com/mediocregopher/mediocre-go-lib/mcmp"
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
func From(cmp *mcmp.Component) *Logger {
	l, _ := cmp.Value(cmpKey(0)).(*Logger)
	if l == nil {
		l = DefaultLogger
	}
	return l
}
