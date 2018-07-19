package mdatastore

import (
	"context"
	. "testing"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mdb"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

var testDS *Datastore

func init() {
	mdb.DefaultGCEProject = "test"
	cfg := mcfg.New()
	testDS = Cfg(cfg)
	cfg.StartTestRun()
}

// Requires datastore emulator to be running
func TestBasic(t *T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

	if _, err := testDS.Put(ctx, key, &val); err != nil {
		t.Fatal(err)
	}

	var val2 valType
	if err := testDS.Get(ctx, key, &val2); err != nil {
		t.Fatal(err)
	}

	massert.Fatal(t, massert.Equal(val, val2))
}
