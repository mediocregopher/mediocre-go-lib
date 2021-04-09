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

// Stacktrace represents a stack trace at a particular point in execution.
type Stacktrace struct {
	frames []uintptr
}

func newStacktrace(skip int) Stacktrace {
	stackSlice := make([]uintptr, MaxStackSize+skip)
	// incr skip once for newStacktrace, and once for runtime.Callers
	l := runtime.Callers(skip+2, stackSlice)
	return Stacktrace{frames: stackSlice[:l]}
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
