package mbigtable

import (
	"context"
	. "testing"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mdb"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

var testBT *Bigtable

func init() {
	mdb.DefaultGCEProject = "test"
	cfg := mcfg.New()
	testBT = Cfg(cfg, "test")
	cfg.StartTestRun()
}

func TestBasic(t *T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tableName := "test-" + mrand.Hex(8)
	colFam := "colFam-" + mrand.Hex(8)
	if err := testBT.EnsureTable(ctx, tableName, colFam); err != nil {
		t.Fatal(err)
	}

	table := testBT.Table(tableName)
	row := "row-" + mrand.Hex(8)
	mut := bigtable.NewMutation()
	mut.Set(colFam, "col", bigtable.Time(time.Now()), []byte("bar"))
	if err := table.Apply(ctx, row, mut); err != nil {
		t.Fatal(err)
	}

	readRow, err := table.ReadRow(ctx, row)
	if err != nil {
		t.Fatal(err)
	}
	readColFam := readRow[colFam]
	massert.Fatal(t, massert.All(
		massert.Len(readColFam, 1),
		massert.Equal(colFam+":col", readColFam[0].Column),
		massert.Equal([]byte("bar"), readColFam[0].Value),
	))
}
