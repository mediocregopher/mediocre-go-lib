package merr

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

// MaxStackSize indicates the maximum number of stack frames which will be
// stored when embedding stack traces in errors.
var MaxStackSize = 50

type stackKey int

// Stacktrace represents a stack trace at a particular point in execution.
type Stacktrace struct {
	frames []uintptr
}

// Frame returns the first frame in the stack.
func (s Stacktrace) Frame() runtime.Frame {
	if len(s.frames) == 0 {
		panic("cannot call Frame on empty stack")
	}

	frame, _ := runtime.CallersFrames([]uintptr(s.frames)).Next()
	return frame
}

// Frames returns all runtime.Frame instances for this stack.
func (s Stacktrace) Frames() []runtime.Frame {
	if len(s.frames) == 0 {
		return nil
	}

	out := make([]runtime.Frame, 0, len(s.frames))
	frames := runtime.CallersFrames([]uintptr(s.frames))
	for {
		frame, more := frames.Next()
		out = append(out, frame)
		if !more {
			break
		}
	}
	return out
}

// String returns a string representing the top-most frame of the stack.
func (s Stacktrace) String() string {
	if len(s.frames) == 0 {
		return ""
	}
	frame := s.Frame()
	file, dir := filepath.Base(frame.File), filepath.Dir(frame.File)
	dir = filepath.Base(dir) // only want the first dirname, ie the pkg name
	return fmt.Sprintf("%s/%s:%d", dir, file, frame.Line)
}

// FullString returns the full stack trace.
func (s Stacktrace) FullString() string {
	sb := new(strings.Builder)
	tw := tabwriter.NewWriter(sb, 0, 4, 4, ' ', 0)
	for _, frame := range s.Frames() {
		file := fmt.Sprintf("%s:%d", frame.File, frame.Line)
		fmt.Fprintf(tw, "%s\t%s\n", file, frame.Function)
	}
	if err := tw.Flush(); err != nil {
		panic(err)
	}
	return sb.String()
}

func setStack(er *err, skip int) {
	stackSlice := make([]uintptr, MaxStackSize+skip)
	// incr skip once for WithStack, and once for runtime.Callers
	l := runtime.Callers(skip+2, stackSlice)
	er.attr[stackKey(0)] = Stacktrace{frames: stackSlice[:l]}
}

// WithStack returns an error with the current stacktrace embedded in it (as a
// Stacktrace type). If skip is non-zero it will skip that many frames from the
// top of the stack. The frame containing the WithStack call itself is always
// excluded.
//
// This call always overwrites any previously existing stack information on the
// error, as opposed to Wrap which only does so if the error didn't already have
// any.
func WithStack(e error, skip int) error {
	er := wrap(e, true)
	setStack(er, skip+1)
	return er
}

func getStack(er *err) (Stacktrace, bool) {
	stack, ok := er.attr[stackKey(0)].(Stacktrace)
	return stack, ok
}

// Stack returns the Stacktrace instance which was embedded by Wrap/WrapSkip, or
// false if none ever was.
func Stack(e error) (Stacktrace, bool) {
	return getStack(wrap(e, false))
}
