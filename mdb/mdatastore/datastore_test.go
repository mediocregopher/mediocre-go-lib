package mdatastore

import (
	. "testing"

	"cloud.google.com/go/datastore"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

// Requires datastore emulator to be running
func TestBasic(t *T) {
	ctx := mtest.NewCtx()
	ctx = mtest.SetEnv(ctx, "GCE_PROJECT", "test")
	ctx, ds := MNew(ctx, nil)
	mtest.Run(ctx, t, func() {
		name := mrand.Hex(8)
		key := datastore.NameKey("testKind", name, nil)
		key.Namespace = "TestBasic_" + mrand.Hex(8)
		type valType struct {
			A, B int
		}
		val := valType{
			A: mrand.Int(),
			B: mrand.Int(),
		}

		if _, err := ds.Put(ctx, key, &val); err != nil {
			t.Fatal(err)
		}

		var val2 valType
		if err := ds.Get(ctx, key, &val2); err != nil {
			t.Fatal(err)
		}

		massert.Fatal(t, massert.Equal(val, val2))
	})
}
