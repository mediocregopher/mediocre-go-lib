package mcfg

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/stretchr/testify/assert"
)

func TestPopulate(t *T) {
	{
		ctx := mctx.New()
		a := Int(ctx, "a", 0, "")
		ctxChild := mctx.ChildOf(ctx, "foo")
		b := Int(ctxChild, "b", 0, "")
		c := Int(ctxChild, "c", 0, "")

		err := Populate(ctx, SourceCLI{
			Args: []string{"--a=1", "--foo-b=2"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, *a)
		assert.Equal(t, 2, *b)
		assert.Equal(t, 0, *c)
	}

	{ // test that required params are enforced
		ctx := mctx.New()
		a := Int(ctx, "a", 0, "")
		ctxChild := mctx.ChildOf(ctx, "foo")
		b := Int(ctxChild, "b", 0, "")
		c := RequiredInt(ctxChild, "c", "")

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
