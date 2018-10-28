package m

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestContext(t *T) {
	ctx := Ctx()
	ctx1 := ChildOf(ctx, "1")
	ctx1a := ChildOf(ctx1, "a")
	ctx1b := ChildOf(ctx1, "b")
	ctx2 := ChildOf(ctx, "2")

	massert.Fatal(t, massert.All(
		massert.Len(Path(ctx), 0),
		massert.Equal(Path(ctx1), []string{"1"}),
		massert.Equal(Path(ctx1a), []string{"1", "a"}),
		massert.Equal(Path(ctx1b), []string{"1", "b"}),
		massert.Equal(Path(ctx2), []string{"2"}),
	))

	massert.Fatal(t, massert.All(
		massert.Equal(
			map[string]context.Context{"1": ctx1, "2": ctx2},
			Children(ctx),
		),
		massert.Equal(
			map[string]context.Context{"a": ctx1a, "b": ctx1b},
			Children(ctx1),
		),
		massert.Equal(
			map[string]context.Context{},
			Children(ctx2),
		),
	))
}
