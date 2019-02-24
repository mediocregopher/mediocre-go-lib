package msql

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest"
)

func TestMySQL(t *T) {
	ctx := mtest.Context()
	ctx, sql := WithMySQL(ctx, "test")
	mtest.Run(ctx, t, func() {
		_, err := sql.Exec("CREATE TABLE IF NOT EXISTS msql_test (id INT);")
		if err != nil {
			t.Fatal(err)
		}
	})
}
