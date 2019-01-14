package merr

import (
	"fmt"
	"runtime"
	"strings"
	"text/tabwriter"
)

// MaxStackSize indicates the maximum number of stack frames which will be
// stored when embedding stack traces in errors.
var MaxStackSize = 50

const attrKeyStack attrKey = "stack"

// Stack represents a stack trace at a particular point in execution.
type Stack []uintptr

// Frame returns the first frame in the stack.
func (s Stack) Frame() runtime.Frame {
	if len(s) == 0 {
		panic("cannot call Frame on empty stack")
	}

	frame, _ := runtime.CallersFrames([]uintptr(s)).Next()
	return frame
}

// Frames returns all runtime.Frame instances for this stack.
func (s Stack) Frames() []runtime.Frame {
	if len(s) == 0 {
		return nil
	}

	out := make([]runtime.Frame, 0, len(s))
	frames := runtime.CallersFrames([]uintptr(s))
	for {
		frame, more := frames.Next()
		out = append(out, frame)
		if !more {
			break
		}
	}
	return out
}

// String returns the full stack trace.
func (s Stack) String() string {
	sb := strBuilderPool.Get().(*strings.Builder)
	defer putStrBuilder(sb)
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
	stackSlice := make([]uintptr, MaxStackSize)
	// incr skip once for setStack, and once for runtime.Callers
	l := runtime.Callers(skip+2, stackSlice)
	er.attr[attrKeyStack] = val{val: Stack(stackSlice[:l]), visible: true}
}

// WithStack returns a copy of the original error, automatically wrapping it if
// the error is not from merr (see Wrap). The returned error has the embedded
// stacktrace set to the frame calling this function.
//
// skip can be used to exclude that many frames from the top of the stack.
func WithStack(e error, skip int) error {
	er := wrap(e, true, -1)
	setStack(er, skip+1)
	return er
}

// GetStack returns the Stack which was embedded in the error, if the error is
// from this package. If not then nil is returned.
func GetStack(e error) Stack {
	stack, _ := wrap(e, false, -1).attr[attrKeyStack].val.(Stack)
	return stack
}
