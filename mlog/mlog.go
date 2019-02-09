// Package mlog is a generic logging library. The log methods come in different
// severities: Debug, Info, Warn, Error, and Fatal.
//
// The log methods take in a message string and a Context. The Context can be
// loaded with additional annotations which will be included in the log entry as
// well (see mctx package).
//
package mlog

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
)

// Truncate is a helper function to truncate a string to a given size. It will
// add 3 trailing elipses, so the returned string will be at most size+3
// characters long
func Truncate(s string, size int) string {
	if len(s) <= size {
		return s
	}
	return s[:size] + "..."
}

////////////////////////////////////////////////////////////////////////////////

// Level describes the severity of a particular log message, and can be compared
// to the severity of any other Level
type Level interface {
	// String gives the string form of the level, e.g. "INFO" or "ERROR"
	String() string

	// Uint gives an integer indicator of the severity of the level, with zero
	// being most severe. If a Level with Uint of zero is logged then the Logger
	// implementation provided by this package will exit the process (i.e. zero
	// is used as Fatal).
	Uint() uint
}

type level struct {
	s string
	i uint
}

func (l level) String() string {
	return l.s
}

func (l level) Uint() uint {
	return l.i
}

// All pre-defined log levels
var (
	DebugLevel Level = level{s: "DEBUG", i: 40}
	InfoLevel  Level = level{s: "INFO", i: 30}
	WarnLevel  Level = level{s: "WARN", i: 20}
	ErrorLevel Level = level{s: "ERROR", i: 10}
	FatalLevel Level = level{s: "FATAL", i: 0}
)

// LevelFromString takes a string describing one of the pre-defined Levels (e.g.
// "debug" or "INFO") and returns the corresponding Level instance, or nil if
// the string doesn't describe any of the predefined Levels.
func LevelFromString(s string) Level {
	switch strings.TrimSpace(strings.ToUpper(s)) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	case "FATAL":
		return FatalLevel
	default:
		return nil
	}
}

////////////////////////////////////////////////////////////////////////////////

// Message describes a message to be logged, after having already resolved the
// KVer
type Message struct {
	Level
	Description string
	Contexts    []context.Context
}

// Handler is a function which can process Messages in some way.
//
// NOTE that Logger does not handle thread-safety, that must be done inside the
// Handler if necessary.
type Handler func(msg Message) error

// MessageJSON is the type used to encode Messages to JSON in DefaultHandler
type MessageJSON struct {
	Level       string `json:"level"`
	Description string `json:"descr"`

	// path -> key -> value
	Annotations map[string]map[string]string `json:"annotations,omitempty"`
}

// DefaultHandler initializes and returns a Handler which will write all
// messages to os.Stderr in a thread-safe way. This is the Handler which
// NewLogger will use automatically.
func DefaultHandler() Handler {
	return defaultHandler(os.Stderr)
}

func defaultHandler(out io.Writer) Handler {
	l := new(sync.Mutex)
	enc := json.NewEncoder(out)
	return func(msg Message) error {
		l.Lock()
		defer l.Unlock()

		msgJSON := MessageJSON{
			Level:       msg.Level.String(),
			Description: msg.Description,
		}
		if len(msg.Contexts) > 0 {
			ctx := mctx.MergeAnnotations(msg.Contexts...)
			msgJSON.Annotations = mctx.Annotations(ctx).StringMapByPath()
		}

		return enc.Encode(msgJSON)
	}
}

// Logger directs Messages to an internal Handler and provides convenient
// methods for creating and modifying its own behavior. All methods are
// thread-safe.
type Logger struct {
	l        *sync.RWMutex
	h        Handler
	maxLevel uint

	testMsgWrittenCh chan struct{} // only initialized/used in tests
}

// NewLogger initializes and returns a new instance of Logger which will write
// to the DefaultHandler.
func NewLogger() *Logger {
	return &Logger{
		l:        new(sync.RWMutex),
		h:        DefaultHandler(),
		maxLevel: InfoLevel.Uint(),
	}
}

// Clone returns an identical instance of the Logger which can be modified
// independently of the original.
func (l *Logger) Clone() *Logger {
	l2 := *l
	l2.l = new(sync.RWMutex)
	return &l2
}

// SetMaxLevel sets the Logger to not log any messages with a higher Level.Uint
// value than of the one given.
func (l *Logger) SetMaxLevel(lvl Level) {
	l.l.Lock()
	defer l.l.Unlock()
	l.maxLevel = lvl.Uint()
}

// SetHandler sets the Logger to use the given Handler in order to process
// Messages.
func (l *Logger) SetHandler(h Handler) {
	l.l.Lock()
	defer l.l.Unlock()
	l.h = h
}

// Handler returns the Handler currently in use by the Logger.
func (l *Logger) Handler() Handler {
	l.l.RLock()
	defer l.l.RUnlock()
	return l.h
}

// Log can be used to manually log a message of some custom defined Level.
//
// If the Level is a fatal (Uint() == 0) then calling this will never return,
// and the process will have os.Exit(1) called.
func (l *Logger) Log(msg Message) {
	l.l.RLock()
	defer l.l.RUnlock()

	if l.maxLevel < msg.Level.Uint() {
		return
	}

	if err := l.h(msg); err != nil {
		go l.Error("Logger.Handler returned error", merr.Context(err))
		return
	}

	if l.testMsgWrittenCh != nil {
		l.testMsgWrittenCh <- struct{}{}
	}

	if msg.Level.Uint() == 0 {
		os.Exit(1)
	}
}

func mkMsg(lvl Level, descr string, ctxs ...context.Context) Message {
	return Message{
		Level:       lvl,
		Description: descr,
		Contexts:    ctxs,
	}
}

// Debug logs a DebugLevel message.
func (l *Logger) Debug(descr string, ctxs ...context.Context) {
	l.Log(mkMsg(DebugLevel, descr, ctxs...))
}

// Info logs a InfoLevel message.
func (l *Logger) Info(descr string, ctxs ...context.Context) {
	l.Log(mkMsg(InfoLevel, descr, ctxs...))
}

// Warn logs a WarnLevel message.
func (l *Logger) Warn(descr string, ctxs ...context.Context) {
	l.Log(mkMsg(WarnLevel, descr, ctxs...))
}

// Error logs a ErrorLevel message.
func (l *Logger) Error(descr string, ctxs ...context.Context) {
	l.Log(mkMsg(ErrorLevel, descr, ctxs...))
}

// Fatal logs a FatalLevel message. A Fatal message automatically stops the
// process with an os.Exit(1)
func (l *Logger) Fatal(descr string, ctxs ...context.Context) {
	l.Log(mkMsg(FatalLevel, descr, ctxs...))
}
