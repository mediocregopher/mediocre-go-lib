package merr

import (
	"strings"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestStack(t *T) {
	foo := New("test")
	fooStack, ok := Stack(foo)
	massert.Require(t, massert.Equal(true, ok))

	// test Frame
	frame := fooStack.Frame()
	massert.Require(t,
		massert.Equal(true, strings.Contains(frame.File, "stack_test.go")),
		massert.Equal(true, strings.Contains(frame.Function, "TestStack")),
	)

	frames := fooStack.Frames()
	massert.Require(t, massert.Comment(
		massert.All(
			massert.Equal(true, len(frames) >= 2),
			massert.Equal(true, strings.Contains(frames[0].File, "stack_test.go")),
			massert.Equal(true, strings.Contains(frames[0].Function, "TestStack")),
		),
		"fooStack.FullString():\n%s", fooStack.FullString(),
	))

	// test that WithStack works and can be used to skip frames
	inner := func() {
		bar := WithStack(foo, 1)
		barStack, _ := Stack(bar)
		frames := barStack.Frames()
		massert.Require(t, massert.Comment(
			massert.All(
				massert.Equal(true, len(frames) >= 2),
				massert.Equal(true, strings.Contains(frames[0].File, "stack_test.go")),
				massert.Equal(true, strings.Contains(frames[0].Function, "TestStack")),
			),
			"barStack.FullString():\n%s", barStack.FullString(),
		))
	}
	inner()

}
