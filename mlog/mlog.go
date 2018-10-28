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
	"bytes"
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

// Message describes a message to be logged, after having already resolved the
// KVer
type Message struct {
	Level
	Msg string
	KV  KV
}

// WriteFn describes a function which formats a single log message and writes it
// to the given io.Writer. If the io.Writer returns an error WriteFn should
// return that error.
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

type msg struct {
	buf *bytes.Buffer
	msg Message
}

// Logger wraps a WriteFn and an io.WriteCloser such that logging calls on the
// Logger will use them (in a thread-safe manner) to write out log messages.
type Logger struct {
	wc       io.WriteCloser
	wfn      WriteFn
	maxLevel uint
	kv       KVer

	msgBufPool       *sync.Pool
	msgCh            chan msg
	testMsgWrittenCh chan struct{} // only initialized/used in tests

	stopCh chan struct{}
	wg     *sync.WaitGroup
}

// NewLogger initializes and returns a new instance of Logger which will write
// to the given WriteCloser.
func NewLogger(wc io.WriteCloser) *Logger {
	l := &Logger{
		wc:  wc,
		wfn: DefaultWriteFn,
		msgBufPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		msgCh:    make(chan msg, 1024),
		maxLevel: InfoLevel.Uint(),
		stopCh:   make(chan struct{}),
		wg:       new(sync.WaitGroup),
	}
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.spin()
	}()
	return l
}

func (l *Logger) cp() *Logger {
	l2 := *l
	return &l2
}

func (l *Logger) drain() {
	for {
		select {
		case m := <-l.msgCh:
			l.writeMsg(m)
		default:
			return
		}
	}
}

func (l *Logger) writeMsg(m msg) {
	if _, err := m.buf.WriteTo(l.wc); err != nil {
		go l.Error("error writing to Logger's WriteCloser", ErrKV(err))
	}
	l.msgBufPool.Put(m.buf)
	if l.testMsgWrittenCh != nil {
		l.testMsgWrittenCh <- struct{}{}
	}
	if m.msg.Level.Uint() == 0 {
		l.wc.Close()
		os.Exit(1)
	}
}

func (l *Logger) spin() {
	defer l.wc.Close()
	for {
		select {
		case m := <-l.msgCh:
			l.writeMsg(m)
		case <-l.stopCh:
			l.drain()
			return
		}
	}
}

// WithMaxLevelUint returns a copy of the Logger with its max logging level set
// to the given uint. The Logger will not log any messages with a higher
// Level.Uint value.
func (l *Logger) WithMaxLevelUint(i uint) *Logger {
	l = l.cp()
	l.maxLevel = i
	return l
}

// WithMaxLevel returns a copy of the Logger with its max Level set to the given
// one. The Logger will not log any messages with a higher Level.Uint value.
func (l *Logger) WithMaxLevel(lvl Level) *Logger {
	return l.WithMaxLevelUint(lvl.Uint())
}

// WithWriteFn returns a copy of the Logger which will use the given WriteFn
// to format and write Messages to the Logger's WriteCloser. This does not
// affect the WriteFn of the original Logger, and both can be used at the same
// time.
func (l *Logger) WithWriteFn(wfn WriteFn) *Logger {
	l = l.cp()
	l.wfn = wfn
	return l
}

// WithKV returns a copy of Logger which will implicitly use the KVers for all
// log messages.
func (l *Logger) WithKV(kvs ...KVer) *Logger {
	l = l.cp()
	l.kv = MergeInto(l.kv, kvs...)
	return l
}

// Stop stops and cleans up any running go-routines and resources held by the
// Logger, allowing it to be garbage-collected. This will flush any remaining
// messages to the io.WriteCloser before returning.
//
// The Logger should not be used after Stop is called
func (l *Logger) Stop() {
	close(l.stopCh)
	l.wg.Wait()
}

// Log can be used to manually log a message of some custom defined Level. kvs
// will be Merge'd automatically. If the Level is a fatal (Uint() == 0) then
// calling this will never return, and the process will have os.Exit(1) called.
func (l *Logger) Log(lvl Level, msgStr string, kvs ...KVer) {
	if l.maxLevel < lvl.Uint() {
		return
	}

	m := Message{Level: lvl, Msg: msgStr, KV: mergeInto(l.kv, kvs...)}
	buf := l.msgBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if err := l.wfn(buf, m); err != nil {
		// TODO welp, hopefully this doesn't infinite loop
		l.Log(ErrorLevel, "Logger could not write to WriteCloser", ErrKV(err))
		return
	}

	select {
	case l.msgCh <- msg{buf: buf, msg: m}:
	case <-l.stopCh:
	}

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
	DefaultLogger = NewLogger(os.Stderr)
	Debug         = DefaultLogger.Debug
	Info          = DefaultLogger.Info
	Warn          = DefaultLogger.Warn
	Error         = DefaultLogger.Error
	Fatal         = DefaultLogger.Fatal
)
