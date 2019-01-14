package merr

import (
	"strings"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestStack(t *T) {
	foo := New("foo")
	fooStack := GetStack(foo)

	// test Frame
	frame := fooStack.Frame()
	massert.Fatal(t, massert.All(
		massert.Equal(true, strings.Contains(frame.File, "stack_test.go")),
		massert.Equal(true, strings.Contains(frame.Function, "TestStack")),
	))

	frames := fooStack.Frames()
	massert.Fatal(t, massert.Comment(
		massert.All(
			massert.Equal(true, len(frames) >= 2),
			massert.Equal(true, strings.Contains(frames[0].File, "stack_test.go")),
			massert.Equal(true, strings.Contains(frames[0].Function, "TestStack")),
		),
		"fooStack.String():\n%s", fooStack.String(),
	))

	// test that WithStack works and can be used to skip frames
	inner := func() {
		bar := WithStack(foo, 1)
		barStack := GetStack(bar)
		frames := barStack.Frames()
		massert.Fatal(t, massert.Comment(
			massert.All(
				massert.Equal(true, len(frames) >= 2),
				massert.Equal(true, strings.Contains(frames[0].File, "stack_test.go")),
				massert.Equal(true, strings.Contains(frames[0].Function, "TestStack")),
			),
			"barStack.String():\n%s", barStack.String(),
		))
	}
	inner()

}
