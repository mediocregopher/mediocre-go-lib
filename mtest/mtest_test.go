package mtest

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
)

func TestRun(t *T) {
	ctx := Context()
	ctx, arg := mcfg.WithRequiredString(ctx, "arg", "Required by this test")
	ctx = WithEnv(ctx, "ARG", "foo")
	Run(ctx, t, func() {
		if *arg != "foo" {
			t.Fatalf(`arg not set to "foo", is set to %q`, *arg)
		}
	})
}
