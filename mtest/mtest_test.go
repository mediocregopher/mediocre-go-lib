package mtest

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
)

func TestRun(t *T) {
	cmp := Component()
	Env(cmp, "ARG", "foo")

	arg := mcfg.String(cmp, "arg", mcfg.ParamRequired())
	Run(cmp, t, func() {
		if *arg != "foo" {
			t.Fatalf(`arg not set to "foo", is set to %q`, *arg)
		}
	})
}
