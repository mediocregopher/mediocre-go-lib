package mbigtable

import (
	. "testing"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestBasic(t *T) {
	ctx := mtest.NewCtx()
	mtest.SetEnv(ctx, "GCE_PROJECT", "testProject")
	bt := MNew(ctx, nil, "testInstance")

	mtest.Run(ctx, t, func() {
		tableName := "test-" + mrand.Hex(8)
		colFam := "colFam-" + mrand.Hex(8)
		if err := bt.EnsureTable(ctx, tableName, colFam); err != nil {
			t.Fatal(err)
		}

		table := bt.Table(tableName)
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
	})
}
