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
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
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

////////////////////////////////////////////////////////////////////////////////

// KVer is used to provide context to a log entry in the form of a dynamic set
// of key/value pairs which can be different for every entry.
//
// Each returned KV should be modifiable.
type KVer interface {
	KV() KV
}

// KVerFunc is a function which implements the KVer interface by calling itself.
type KVerFunc func() KV

// KV implements the KVer interface by calling the KVerFunc itself.
func (kvf KVerFunc) KV() KV {
	return kvf()
}

// KV is a set of key/value pairs which provides context for a log entry by a
// KVer. KV is itself also a KVer.
type KV map[string]interface{}

// KV implements the KVer method by returning a copy of the KV
func (kv KV) KV() KV {
	nkv := make(KV, len(kv))
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

// this may take in any amount of nil values, but should never return nil
func mergeInto(kv KVer, kvs ...KVer) KV {
	if kv == nil {
		kv = KV(nil) // will return empty map when KV is called on it
	}
	kvm := kv.KV()
	for _, kv := range kvs {
		if kv == nil {
			continue
		}
		for k, v := range kv.KV() {
			kvm[k] = v
		}
	}
	return kvm
}

type merger struct {
	base KVer
	rest []KVer
}

// Merge takes in multiple KVers and returns a single KVer which is the union of
// all the passed in ones. Key/Vals on the rightmost of the set take precedence
// over conflicting ones to the left.
//
// The KVer returned will call KV() on each of the passed in KVers every time
// its KV method is called.
func Merge(kvs ...KVer) KVer {
	if len(kvs) == 0 {
		return merger{}
	}
	return merger{base: kvs[0], rest: kvs[1:]}
}

// MergeInto is a convenience function which acts similarly to Merge.
func MergeInto(kv KVer, kvs ...KVer) KVer {
	return merger{base: kv, rest: kvs}
}

func (m merger) KV() KV {
	return mergeInto(m.base, m.rest...)
}

// Prefix prefixes the all keys returned from the given KVer with the given
// prefix string.
func Prefix(kv KVer, prefix string) KVer {
	return KVerFunc(func() KV {
		kvv := kv.KV()
		newKVV := make(KV, len(kvv))
		for k, v := range kvv {
			newKVV[prefix+k] = v
		}
		return newKVV
	})
}

////////////////////////////////////////////////////////////////////////////////

// Message describes a message to be logged, after having already resolved the
// KVer
type Message struct {
	Level
	Msg string
	KV  KV
}

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

// Handler is a function which can process Messages in some way.
//
// NOTE that Logger does not handle thread-safety, that must be done inside the
// Handler if necessary.
type Handler func(msg Message) error

// DefaultFormat formats and writs the Message to the given Writer using mlog's
// default format.
func DefaultFormat(w io.Writer, msg Message) error {
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

// DefaultHandler initializes and returns a Handler which will write all
// messages to os.Stderr in a thread-safe way. This is the Handler which
// NewLogger will use automatically.
func DefaultHandler() Handler {
	l := new(sync.Mutex)
	bw := bufio.NewWriter(os.Stderr)
	return func(msg Message) error {
		l.Lock()
		defer l.Unlock()

		err := DefaultFormat(bw, msg)
		if err != nil {
			err = bw.Flush()
		}
		return err
	}
}

// Logger directs Messages to an internal Handler and provides convenient
// methods for creating and modifying its own behavior.
type Logger struct {
	w        io.Writer
	h        Handler
	maxLevel uint

	testMsgWrittenCh chan struct{} // only initialized/used in tests
}

// NewLogger initializes and returns a new instance of Logger which will write
// to the DefaultHandler.
func NewLogger() *Logger {
	return &Logger{
		h:        DefaultHandler(),
		maxLevel: InfoLevel.Uint(),
	}
}

// Handler returns the Handler currently in use by the Logger.
func (l *Logger) Handler() Handler {
	return l.h
}

func (l *Logger) clone() *Logger {
	l2 := *l
	return &l2
}

// WithMaxLevelUint returns a copy of the Logger which will not log any messages
// with a higher Level.Uint value than the one given. The returned Logger is
// identical in all other aspects.
func (l *Logger) WithMaxLevelUint(i uint) *Logger {
	l = l.clone()
	l.maxLevel = i
	return l
}

// WithMaxLevel returns a copy of the Logger which will not log any messages
// with a higher Level.Uint value than of the Level given. The returned Logger
// is identical in all other aspects.
func (l *Logger) WithMaxLevel(lvl Level) *Logger {
	return l.WithMaxLevelUint(lvl.Uint())
}

// WithHandler returns a copy of the Logger which will use the given Handler in
// order to process messages. The returned Logger is identical in all other
// aspects.
func (l *Logger) WithHandler(h Handler) *Logger {
	l = l.clone()
	l.h = h
	return l
}

// Log can be used to manually log a message of some custom defined Level. kvs
// will be Merge'd automatically. If the Level is a fatal (Uint() == 0) then
// calling this will never return, and the process will have os.Exit(1) called.
func (l *Logger) Log(lvl Level, msgStr string, kvs ...KVer) {
	if l.maxLevel < lvl.Uint() {
		return
	}

	m := Message{Level: lvl, Msg: msgStr, KV: Merge(kvs...).KV()}
	if err := l.h(m); err != nil {
		go l.Log(ErrorLevel, "Logger.Handler returned error", ErrKV(err))
		return
	}

	if l.testMsgWrittenCh != nil {
		l.testMsgWrittenCh <- struct{}{}
	}

	if m.Level.Uint() == 0 {
		os.Exit(1)
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
