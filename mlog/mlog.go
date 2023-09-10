// Package mlog is a generic logging library. The log methods come in different
// severities: Debug, Info, Warn, Error, and Fatal.
//
// The log methods take in a message string and a Context. The Context can be
// loaded with additional annotations which will be included in the log entry as
// well (see mctx package).
package mlog

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/v2/mctx"
	"github.com/mediocregopher/mediocre-go-lib/v2/merr"
)

type mlogAnnotation string

// Null is an instance of Logger which will write all Messages to /dev/null.
var Null = NewLogger(&LoggerOpts{
	MessageHandler: NewMessageHandler(ioutil.Discard),
})

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

	// Int gives an integer indicator of the severity of the level, with zero
	// being most severe. If a Level with a negative Int is logged then the
	// Logger implementation provided by this package will exit the process.
	Int() int
}

type level struct {
	s string
	i int
}

func (l level) String() string {
	return l.s
}

func (l level) Int() int {
	return l.i
}

// All pre-defined log levels
var (
	LevelDebug Level = level{s: "DEBUG", i: 40}
	LevelInfo  Level = level{s: "INFO", i: 30}
	LevelWarn  Level = level{s: "WARN", i: 20}
	LevelError Level = level{s: "ERROR", i: 10}
	LevelFatal Level = level{s: "FATAL", i: -1}
)

// LevelFromString takes a string describing one of the pre-defined Levels (e.g.
// "debug" or "INFO") and returns the corresponding Level instance, or nil if
// the string doesn't describe any of the predefined Levels.
func LevelFromString(s string) Level {
	switch strings.TrimSpace(strings.ToUpper(s)) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return nil
	}
}

////////////////////////////////////////////////////////////////////////////////

// Message describes a message to be logged.
type Message struct {
	Context context.Context
	Level
	Description string
}

// FullMessage extends Message to contain loggable properties not provided
// directly by the user.
type FullMessage struct {
	Message
	Time      time.Time
	Namespace []string
}

// LoggerOpts are optional parameters to NewLogger. All fields are optional. A
// nil value of LoggerOpts is equivalent to an empty one.
type LoggerOpts struct {
	// MessageHandler is the MessageHandler which will be used to process
	// Messages.
	//
	// Defaults to NewMessageHandler(os.Stderr).
	MessageHandler MessageHandler

	// MaxLevel indicates the maximum log level which should be handled. See the
	// Level interface for more.
	//
	// Defaults to LevelInfo.Int().
	MaxLevel int

	// Now returns the current time.Time whenever it is called.
	//
	// Defaults to time.Now.
	Now func() time.Time
}

func (o *LoggerOpts) withDefaults() *LoggerOpts {
	out := new(LoggerOpts)
	if o != nil {
		*out = *o
	}

	if out.MessageHandler == nil {
		out.MessageHandler = NewMessageHandler(os.Stderr)
	}

	if out.MaxLevel == 0 {
		out.MaxLevel = LevelInfo.Int()
	}

	if out.Now == nil {
		out.Now = time.Now
	}

	return out
}

// Logger creates and directs Messages to an internal MessageHandler. All
// methods are thread-safe.
type Logger struct {
	opts *LoggerOpts
	l    *sync.RWMutex
	ns   []string
}

// NewLogger initializes and returns a new instance of Logger.
func NewLogger(opts *LoggerOpts) *Logger {
	return &Logger{
		opts: opts.withDefaults(),
		l:    new(sync.RWMutex),
	}
}

// Close cleans up all resources held by the Logger.
func (l *Logger) Close() error {
	if err := l.opts.MessageHandler.Sync(); err != nil {
		return err
	}
	return nil
}

func (l *Logger) clone() *Logger {

	l2 := &Logger{
		opts: &LoggerOpts{
			MessageHandler: l.opts.MessageHandler,
			MaxLevel:       l.opts.MaxLevel,
			Now:            l.opts.Now,
		},
		l:  new(sync.RWMutex),
		ns: make([]string, len(l.ns), len(l.ns)+1),
	}

	copy(l2.ns, l.ns)
	return l2
}

// WithNamespace returns a clone of the Logger with the given value appended to
// its namespace array. The namespace array is included in every FullMessage
// which is handled by Logger's MessageHandler.
func (l *Logger) WithNamespace(name string) *Logger {
	l = l.clone()
	l.ns = append(l.ns, name)
	return l
}

// WithMaxLevel returns a clone of the Logger with the given MaxLevel set as the
// value for the maximum log level which will be output (see MaxLevel in
// LoggerOpts).
func (l *Logger) WithMaxLevel(level int) *Logger {
	l = l.clone()
	l.opts.MaxLevel = level
	return l
}

// MaxLevel returns the Logger's maximum level which it will log (see MaxLevel
// in LoggerOpts, and the WithMaxLevel method).
func (l *Logger) MaxLevel() int {
	return l.opts.MaxLevel
}

// Log can be used to manually log a message of some custom defined Level.
//
// If the Level is a fatal (Int() < 0) then calling this will never return,
// and the process will have os.Exit(1) called.
func (l *Logger) Log(msg Message) {
	l.l.RLock()
	defer l.l.RUnlock()

	if l.opts.MaxLevel < msg.Level.Int() {
		return
	}

	fullMsg := FullMessage{
		Message:   msg,
		Time:      l.opts.Now(),
		Namespace: l.ns,
	}

	if err := l.opts.MessageHandler.Handle(fullMsg); err != nil {
		go l.Error(context.Background(), "MessageHandler.Handle returned error", err)
		return
	}

	if msg.Level.Int() < 0 {
		l.opts.MessageHandler.Sync()
		os.Exit(1)
	}
}

func mkMsg(ctx context.Context, lvl Level, descr string) Message {
	return Message{
		Context:     ctx,
		Level:       lvl,
		Description: descr,
	}
}

func mkErrMsg(ctx context.Context, lvl Level, descr string, err error) Message {
	var e merr.Error
	if !errors.As(err, &e) {
		ctx = mctx.Annotate(ctx, mlogAnnotation("errMsg"), err.Error())
		return mkMsg(ctx, lvl, descr)
	}

	ctx = mctx.Annotate(ctx,
		mlogAnnotation("errMsg"), err.Error(),
		mlogAnnotation("errCtx"), mctx.ContextAsAnnotator(e.Ctx),
		mlogAnnotation("errLine"), e.Stacktrace.String(),
	)

	return mkMsg(ctx, lvl, descr)
}

// Debug logs a LevelDebug message.
func (l *Logger) Debug(ctx context.Context, descr string) {
	l.Log(mkMsg(ctx, LevelDebug, descr))
}

// Info logs a LevelInfo message.
func (l *Logger) Info(ctx context.Context, descr string) {
	l.Log(mkMsg(ctx, LevelInfo, descr))
}

// WarnString logs a LevelWarn message which is only a string.
func (l *Logger) WarnString(ctx context.Context, descr string) {
	l.Log(mkMsg(ctx, LevelWarn, descr))
}

// Warn logs a LevelWarn message, including information from the given error.
func (l *Logger) Warn(ctx context.Context, descr string, err error) {
	l.Log(mkErrMsg(ctx, LevelWarn, descr, err))
}

// ErrorString logs a LevelError message which is only a string.
func (l *Logger) ErrorString(ctx context.Context, descr string) {
	l.Log(mkMsg(ctx, LevelError, descr))
}

// Error logs a LevelError message, including information from the given error.
func (l *Logger) Error(ctx context.Context, descr string, err error) {
	l.Log(mkErrMsg(ctx, LevelError, descr, err))
}

// FatalString logs a LevelFatal message which is only a string. A Fatal message
// automatically stops the process with an os.Exit(1) if the default
// MessageHandler is used.
func (l *Logger) FatalString(ctx context.Context, descr string) {
	l.Log(mkMsg(ctx, LevelFatal, descr))
}

// Fatal logs a LevelFatal message. A Fatal message automatically stops the
// process with an os.Exit(1) if the default MessageHandler is used.
func (l *Logger) Fatal(ctx context.Context, descr string, err error) {
	l.Log(mkErrMsg(ctx, LevelFatal, descr, err))
}
