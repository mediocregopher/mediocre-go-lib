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
		ctx, a := WithInt(ctx, "a", 0, "")
		ctxChild := mctx.NewChild(ctx, "foo")
		ctxChild, b := WithInt(ctxChild, "b", 0, "")
		ctxChild, c := WithInt(ctxChild, "c", 0, "")
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
		ctx, a := WithInt(ctx, "a", 0, "")
		ctxChild := mctx.NewChild(ctx, "foo")
		ctxChild, b := WithInt(ctxChild, "b", 0, "")
		ctxChild, c := WithRequiredInt(ctxChild, "c", "")
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
