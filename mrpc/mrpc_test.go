package mrpc

import (
	"context"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/stretchr/testify/assert"
)

func TestReflectClient(t *T) {
	type argT struct {
		In string
	}

	type resT struct {
		Out string
	}

	ctx := context.Background()

	{ // test with handler returning non-pointer
		client := ReflectClient(HandlerFunc(func(c Call) (interface{}, error) {
			var args argT
			assert.NoError(t, c.UnmarshalArgs(&args))
			assert.Equal(t, "foo", c.Method())
			return resT{Out: args.In}, nil
		}))

		{ // test with arg being non-pointer
			in := mtest.RandHex(8)
			var res resT
			assert.NoError(t, client.CallRPC(ctx, &res, "foo", argT{In: in}))
			assert.Equal(t, in, res.Out)
		}

		{ // test with arg being pointer
			in := mtest.RandHex(8)
			var res resT
			assert.NoError(t, client.CallRPC(ctx, &res, "foo", &argT{In: in}))
			assert.Equal(t, in, res.Out)
		}
	}

	{ // test with handler returning pointer
		client := ReflectClient(HandlerFunc(func(c Call) (interface{}, error) {
			var args argT
			assert.NoError(t, c.UnmarshalArgs(&args))
			assert.Equal(t, "foo", c.Method())
			return &resT{Out: args.In}, nil
		}))

		{ // test with arg being non-pointer
			in := mtest.RandHex(8)
			var res resT
			assert.NoError(t, client.CallRPC(ctx, &res, "foo", argT{In: in}))
			assert.Equal(t, in, res.Out)
		}

		{ // test with arg being pointer
			in := mtest.RandHex(8)
			var res resT
			assert.NoError(t, client.CallRPC(ctx, &res, "foo", &argT{In: in}))
			assert.Equal(t, in, res.Out)
		}
	}
}
