package merr

import (
	"errors"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestError(t *T) {
	er := &err{
		err: errors.New("foo"),
		attr: map[interface{}]val{
			"a":   val{val: "aaa aaa\n", visible: true},
			"b":   val{val: "invisible"},
			"c":   val{val: "ccc\nccc\n", visible: true},
			"d\t": val{val: "weird key but ok", visible: true},
		},
	}
	str := er.Error()
	exp := `foo
	* a: aaa aaa
	* c: 
		ccc
		ccc
	* d\t: weird key but ok`
	massert.Fatal(t, massert.Equal(exp, str))
}

func TestBase(t *T) {
	errFoo, errBar := errors.New("foo"), errors.New("bar")
	erFoo := Wrap(errFoo)
	massert.Fatal(t, massert.All(
		massert.Equal(errFoo, Base(erFoo)),
		massert.Equal(errBar, Base(errBar)),
		massert.Not(massert.Equal(errFoo, erFoo)),
		massert.Not(massert.Equal(errBar, Base(erFoo))),
		massert.Equal(true, Equal(errFoo, erFoo)),
		massert.Equal(false, Equal(errBar, erFoo)),
	))
}
