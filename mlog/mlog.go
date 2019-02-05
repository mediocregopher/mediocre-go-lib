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
//	log := mlog.NewLogger()
//	log.Info("Something important has occurred")
//	log.Error("Could not open file", mlog.KV{"filename": filename}, merr.KV(err))
//
package mlog

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
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

// KVer is used to provide context to a log entry in the form of a dynamic set
// of key/value pairs which can be different for every entry.
//
// The returned map is read-only, and may be nil.
type KVer interface {
	KV() map[string]interface{}
}

// KVerFunc is a function which implements the KVer interface by calling itself.
type KVerFunc func() map[string]interface{}

// KV implements the KVer interface by calling the KVerFunc itself.
func (kvf KVerFunc) KV() map[string]interface{} {
	return kvf()
}

// KV is a KVer which returns a copy of itself when KV is called.
type KV map[string]interface{}

// KV implements the KVer method by returning a copy of itself.
func (kv KV) KV() map[string]interface{} {
	return map[string]interface{}(kv)
}

// Set returns a copy of the KV being called on with the given key/val set on
// it. The original KV is unaffected
func (kv KV) Set(k string, v interface{}) KV {
	kvm := make(map[string]interface{}, len(kv)+1)
	copyM(kvm, kv.KV())
	kvm[k] = v
	return KV(kvm)
}

// returns a key/value map which should not be written to. saves a map-cloning
// if KVer is a KV
func readOnlyKVM(kver KVer) map[string]interface{} {
	if kver == nil {
		return map[string]interface{}(nil)
	} else if kv, ok := kver.(KV); ok {
		return map[string]interface{}(kv)
	}
	return kver.KV()
}

func copyM(dst, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

// this may take in any amount of nil values, but should never return nil
func mergeInto(kv KVer, kvs ...KVer) map[string]interface{} {
	kvm := map[string]interface{}{}
	if kv != nil {
		copyM(kvm, kv.KV())
	}
	for _, innerKV := range kvs {
		if innerKV == nil {
			continue
		}
		copyM(kvm, innerKV.KV())
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

func (m merger) KV() map[string]interface{} {
	return mergeInto(m.base, m.rest...)
}

// Prefix prefixes all keys returned from the given KVer with the given prefix
// string.
func Prefix(kv KVer, prefix string) KVer {
	return KVerFunc(func() map[string]interface{} {
		kvm := readOnlyKVM(kv)
		newKVM := make(map[string]interface{}, len(kvm))
		for k, v := range kvm {
			newKVM[prefix+k] = v
		}
		return newKVM
	})
}

////////////////////////////////////////////////////////////////////////////////

// Message describes a message to be logged, after having already resolved the
// KVer
type Message struct {
	context.Context
	Level
	Description string
	KVer
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

// DefaultFormat formats and writes the Message to the given Writer using mlog's
// default format.
func DefaultFormat(w io.Writer, msg Message) error {
	var err error
	write := func(s string, args ...interface{}) {
		if err == nil {
			_, err = fmt.Fprintf(w, s, args...)
		}
	}
	write("~ %s -- ", msg.Level.String())
	if path := mctx.Path(msg.Context); len(path) > 0 {
		write("(%s) ", "/"+strings.Join(path, "/"))
	}
	write("%s", msg.Description)
	if msg.KVer != nil {
		if kv := msg.KV(); len(kv) > 0 {
			write(" --")
			for _, kve := range stringSlice(kv) {
				write(" %s=%s", kve[0], kve[1])
			}
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
		if err == nil {
			err = bw.Flush()
		}
		return err
	}
}

// Logger directs Messages to an internal Handler and provides convenient
// methods for creating and modifying its own behavior. All methods are
// thread-safe.
type Logger struct {
	l        *sync.RWMutex
	h        Handler
	maxLevel uint
	kv       KVer

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

// SetKV sets the Logger to use the merging of the given KVers as a base KVer
// for all Messages. If the Logger already had a base KVer (via a previous SetKV
// call) then this set will be merged onto that one.
func (l *Logger) SetKV(kvs ...KVer) {
	l.l.Lock()
	defer l.l.Unlock()
	l.kv = MergeInto(l.kv, kvs...)
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

	if l.kv != nil {
		msg.KVer = MergeInto(l.kv, msg.KVer)
	}

	if err := l.h(msg); err != nil {
		go l.Error(context.Background(), "Logger.Handler returned error", merr.KV(err))
		return
	}

	if l.testMsgWrittenCh != nil {
		l.testMsgWrittenCh <- struct{}{}
	}

	if msg.Level.Uint() == 0 {
		os.Exit(1)
	}
}

func mkMsg(ctx context.Context, lvl Level, descr string, kvs ...KVer) Message {
	return Message{
		Context:     ctx,
		Level:       lvl,
		Description: descr,
		KVer:        Merge(kvs...),
	}
}

// Debug logs a DebugLevel message, merging the KVers together first
func (l *Logger) Debug(ctx context.Context, descr string, kvs ...KVer) {
	l.Log(mkMsg(ctx, DebugLevel, descr, kvs...))
}

// Info logs a InfoLevel message, merging the KVers together first
func (l *Logger) Info(ctx context.Context, descr string, kvs ...KVer) {
	l.Log(mkMsg(ctx, InfoLevel, descr, kvs...))
}

// Warn logs a WarnLevel message, merging the KVers together first
func (l *Logger) Warn(ctx context.Context, descr string, kvs ...KVer) {
	l.Log(mkMsg(ctx, WarnLevel, descr, kvs...))
}

// Error logs a ErrorLevel message, merging the KVers together first
func (l *Logger) Error(ctx context.Context, descr string, kvs ...KVer) {
	l.Log(mkMsg(ctx, ErrorLevel, descr, kvs...))
}

// Fatal logs a FatalLevel message, merging the KVers together first. A Fatal
// message automatically stops the process with an os.Exit(1)
func (l *Logger) Fatal(ctx context.Context, descr string, kvs ...KVer) {
	l.Log(mkMsg(ctx, FatalLevel, descr, kvs...))
}
