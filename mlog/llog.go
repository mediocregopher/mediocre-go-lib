// Package mlog is a generic logging library. The log methods come in different
// severities: Debug, Info, Warn, Error, and Fatal.
//
// The log methods take in a string describing the error, and a set of key/value
// pairs giving the specific context around the error. The string is intended to
// always be the same no matter what, while the key/value pairs give information
// like which userID the error happened to, or any other relevant contextual
// information.
//
// Examples:
//
//	Info("Something important has occurred")
//	Error("Could not open file", llog.KV{"filename": filename}, llog.ErrKV(err))
//
package mlog

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// LevelFromString parses a string, possibly lowercase, and returns the Level
// identified by it, or an error.
//
// Note that this only works for the Levels pre-defined in this package, if
// you've extended the package to use your own levels you'll have to implement
// your own LevelFromString method.
func LevelFromString(ls string) (Level, error) {
	var l Level
	switch strings.ToUpper(ls) {
	case "DEBUG":
		l = DebugLevel
	case "INFO":
		l = InfoLevel
	case "WARN":
		l = WarnLevel
	case "ERROR":
		l = ErrorLevel
	case "FATAL":
		l = FatalLevel
	default:
		return nil, fmt.Errorf("unknown log level %q", ls)
	}

	return l, nil
}

// KVer is used to provide context to a log entry in the form of a dynamic set
// of key/value pairs which can be different for every entry.
//
// Each returned map should be modifiable.
type KVer interface {
	KV() map[string]interface{}
}

// KV is a simple and convenient implementation of KVer
type KV map[string]interface{}

// KV implements the KVer method by returning a copy of the casted KV
func (kv KV) KV() map[string]interface{} {
	nkv := make(map[string]interface{}, len(kv))
	for k, v := range kv {
		nkv[k] = v
	}
	return nkv
}

// Set returns a copy of the KV being called on with the given key/val set on
// it. The original KV is unaffected
func (kv KV) Set(k string, v interface{}) KV {
	nkv := kv.KV()
	nkv[k] = v
	return nkv
}

// separate from Merge because it's convenient to not return a KVer if that KVer
// is going to immediately have KV called on it (and thereby create a copy for
// no reason).
func merge(kvs ...KVer) KV {
	kvm := kvs[0].KV()
	for _, kv := range kvs[1:] {
		for k, v := range kv.KV() {
			kvm[k] = v
		}
	}
	return KV(kvm)
}

// Merge takes in multiple KVers and returns a single KVer which is the union of
// all the passed in ones. Key/vals on the rightmost of the set take precedence
// over conflicting ones to the left.
func Merge(kvs ...KVer) KVer {
	return merge(kvs...)
}

// Message describes a message to be logged, after having already resolved the
// KVer
type Message struct {
	Level
	Msg string
	KV  KV
}

// WriteFn describes a function which formats a single log message and writes it
// to the given io.Writer. If the io.Writer returns an error this should return
// that error.
type WriteFn func(w io.Writer, msg Message) error

func stringSlice(kv KV) [][2]string {
	slice := make([][2]string, 0, len(kv))
	for k, v := range kv {
		slice = append(slice, [2]string{
			k,
			strconv.QuoteToGraphic(fmt.Sprint(v)),
		})
	}
	sort.Slice(slice, func(i, j int) bool {
		return slice[i][0] < slice[j][0]
	})
	return slice
}

// DefaultWriteFn is the default implementation of WriteFn.
func DefaultWriteFn(w io.Writer, msg Message) error {
	var err error
	write := func(s string, args ...interface{}) {
		if err == nil {
			_, err = fmt.Fprintf(w, s, args...)
		}
	}
	write("~ %s -- %s", msg.Level.String(), msg.Msg)
	if len(msg.KV) > 0 {
		write(" --")
		for _, kve := range stringSlice(msg.KV) {
			write(" %s=%s", kve[0], kve[1])
		}
	}
	write("\n")
	return err
}

// Logger wraps a WriteFn and an io.WriteCloser such that logging calls on the
// Logger will use them (in a thread-safe manner) to write out log messages.
//
// Logger will auto-initialize itself on the first method call.
type Logger struct {
	WriteFn        // Defaults to DefaultWriteFn
	io.WriteCloser // Defaults to os.Stderr

	levelL      sync.RWMutex
	maxLevel    uint
	maxLevelSet bool

	init             sync.Once
	msgCh            chan Message
	testMsgWrittenCh chan struct{} // only initialized/used in tests
}

func (l *Logger) getMaxLevel() uint {
	l.levelL.RLock()
	defer l.levelL.RUnlock()
	if !l.maxLevelSet {
		return InfoLevel.Uint()
	}
	return l.maxLevel
}

// SetMaxLevelUint sets the maximum (up-to-and-including) level priority which
// the Logger will output a log for. Any Levels with an Uint value higher than
// this will not be logged. This may be called at any time.
func (l *Logger) SetMaxLevelUint(i uint) {
	l.levelL.Lock()
	l.maxLevel = i
	l.maxLevelSet = true
	l.levelL.Unlock()
}

// SetMaxLevel sets the maximum (up-to-and-including) Level which the Logger
// will output a log for. Any Levels whose Uint value is higher than this one's
// will not be logged. This may be called at any time.
func (l *Logger) SetMaxLevel(lvl Level) {
	l.SetMaxLevelUint(lvl.Uint())
}

func (l *Logger) spin() {
	for msg := range l.msgCh {
		if err := l.WriteFn(l.WriteCloser, msg); err != nil {
			go l.Error("could not write to Logger.WriteCloser", ErrKV(err))
		}
		if l.testMsgWrittenCh != nil {
			l.testMsgWrittenCh <- struct{}{}
		}
		if msg.Level.Uint() == 0 {
			l.WriteCloser.Close()
			os.Exit(1)
		}
	}
	l.WriteCloser.Close()
}

func (l *Logger) initDo() {
	l.init.Do(func() {
		if l.WriteFn == nil {
			l.WriteFn = DefaultWriteFn
		}
		if l.WriteCloser == nil {
			l.WriteCloser = os.Stderr
		}
		// maxLevel's default is implicitly handled by getMaxLevel
		l.msgCh = make(chan Message, 1000)
		go l.spin()
	})
}

// Stop stops and cleans up any running go-routines and resources held by the
// Logger, allowing it to be garbage-collected. The Logger should not be used
// after Stop is called
func (l *Logger) Stop() {
	l.initDo()
	close(l.msgCh)
}

// Log can be used to manually log a message of some custom defined Level. kvs
// will be Merge'd automatically. If the Level is a fatal (Uint() == 0) then
// calling this will never return, and the process will have os.Exit(1) called.
func (l *Logger) Log(lvl Level, msg string, kvs ...KVer) {
	if maxLevel := l.getMaxLevel(); maxLevel < lvl.Uint() {
		return
	}

	var kv KV
	if len(kvs) > 0 {
		kv = merge(kvs...)
	}

	l.initDo()
	l.msgCh <- Message{Level: lvl, Msg: msg, KV: kv}

	// if a Fatal is logged then we're merely waiting here for spin to call
	// os.Exit, and this go-routine shouldn't be allowed to continue
	if lvl.Uint() == 0 {
		select {}
	}
}

// Debug logs a DebugLevel message, merging the KVers together first
func (l *Logger) Debug(msg string, kvs ...KVer) {
	l.Log(DebugLevel, msg, kvs...)
}

// Info logs a InfoLevel message, merging the KVers together first
func (l *Logger) Info(msg string, kvs ...KVer) {
	l.Log(InfoLevel, msg, kvs...)
}

// Warn logs a WarnLevel message, merging the KVers together first
func (l *Logger) Warn(msg string, kvs ...KVer) {
	l.Log(WarnLevel, msg, kvs...)
}

// Error logs a ErrorLevel message, merging the KVers together first
func (l *Logger) Error(msg string, kvs ...KVer) {
	l.Log(ErrorLevel, msg, kvs...)
}

// Fatal logs a FatalLevel message, merging the KVers together first. A Fatal
// message automatically stops the process with an os.Exit(1)
func (l *Logger) Fatal(msg string, kvs ...KVer) {
	l.Log(FatalLevel, msg, kvs...)
}

// DefaultLogger is a Logger using the default configuration (which will log to
// stderr). The Debug, Info, Warn, etc... methods from DefaultLogger are exposed
// as global functions for convenience. Because Logger is not truly initialized
// till the first time it is called any of DefaultLogger's fields may be
// modified before using one of the Debug, Info, Warn, etc... global functions.
var (
	DefaultLogger = new(Logger)
	Debug         = DefaultLogger.Debug
	Info          = DefaultLogger.Info
	Warn          = DefaultLogger.Warn
	Error         = DefaultLogger.Error
	Fatal         = DefaultLogger.Fatal
)
