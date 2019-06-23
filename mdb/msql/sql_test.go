package msql

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest"
)

func TestMySQL(t *T) {
	cmp := mtest.Component()
	sql := InstMySQL(cmp, "test")
	mtest.Run(cmp, t, func() {
		_, err := sql.Exec("CREATE TABLE IF NOT EXISTS msql_test (id INT);")
		if err != nil {
			t.Fatal(err)
		}
	})
}
