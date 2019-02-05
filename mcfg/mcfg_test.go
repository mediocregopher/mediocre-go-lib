package mcfg

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/stretchr/testify/assert"
)

func TestPopulate(t *T) {
	{
		ctx := context.Background()
		ctx, a := Int(ctx, "a", 0, "")
		ctxChild := mctx.NewChild(ctx, "foo")
		ctxChild, b := Int(ctxChild, "b", 0, "")
		ctxChild, c := Int(ctxChild, "c", 0, "")
		ctx = mctx.WithChild(ctx, ctxChild)

		err := Populate(ctx, SourceCLI{
			Args: []string{"--a=1", "--foo-b=2"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, *a)
		assert.Equal(t, 2, *b)
		assert.Equal(t, 0, *c)
	}

	{ // test that required params are enforced
		ctx := context.Background()
		ctx, a := Int(ctx, "a", 0, "")
		ctxChild := mctx.NewChild(ctx, "foo")
		ctxChild, b := Int(ctxChild, "b", 0, "")
		ctxChild, c := RequiredInt(ctxChild, "c", "")
		ctx = mctx.WithChild(ctx, ctxChild)

		err := Populate(ctx, SourceCLI{
			Args: []string{"--a=1", "--foo-b=2"},
		})
		assert.Error(t, err)

		err = Populate(ctx, SourceCLI{
			Args: []string{"--a=1", "--foo-b=2", "--foo-c=3"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, *a)
		assert.Equal(t, 2, *b)
		assert.Equal(t, 3, *c)
	}
}
