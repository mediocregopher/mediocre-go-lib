package merr

import (
	"strings"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestKV(t *T) {
	massert.Fatal(t, massert.All(
		massert.Nil(WithValue(nil, "foo", "bar", true)),
		massert.Nil(WithValue(nil, "foo", "bar", false)),
		massert.Nil(GetValue(nil, "foo")),
		massert.Len(KV(nil).KV(), 0),
	))

	er := New("foo")
	kv := KV(er).KV()
	massert.Fatal(t, massert.Comment(
		massert.All(
			massert.Len(kv, 2),
			massert.Equal("foo", kv["err"]),
			massert.Equal(true,
				strings.HasPrefix(kv["errSrc"].(string), "merr/kv_test.go:")),
		),
		"kv: %#v", kv,
	))

	type A string
	type B string
	type C string

	er = WithValue(er, "invisible", "you can't see me", false)
	er = WithValue(er, A("k"), "1", true)
	kv = KV(er).KV()
	massert.Fatal(t, massert.Comment(
		massert.All(
			massert.Len(kv, 3),
			massert.Equal("foo", kv["err"]),
			massert.Equal(true,
				strings.HasPrefix(kv["errSrc"].(string), "merr/kv_test.go:")),
			massert.Equal("1", kv["k"]),
		),
		"kv: %#v", kv,
	))

	er = WithValue(er, B("k"), "2", true)
	kv = KV(er).KV()
	massert.Fatal(t, massert.Comment(
		massert.All(
			massert.Len(kv, 4),
			massert.Equal("foo", kv["err"]),
			massert.Equal(true,
				strings.HasPrefix(kv["errSrc"].(string), "merr/kv_test.go:")),
			massert.Equal("1", kv["merr.A(k)"]),
			massert.Equal("2", kv["merr.B(k)"]),
		),
		"kv: %#v", kv,
	))

	er = WithValue(er, C("k"), "3", true)
	kv = KV(er).KV()
	massert.Fatal(t, massert.Comment(
		massert.All(
			massert.Len(kv, 5),
			massert.Equal("foo", kv["err"]),
			massert.Equal(true,
				strings.HasPrefix(kv["errSrc"].(string), "merr/kv_test.go:")),
			massert.Equal("1", kv["merr.A(k)"]),
			massert.Equal("2", kv["merr.B(k)"]),
			massert.Equal("3", kv["merr.C(k)"]),
		),
		"kv: %#v", kv,
	))
}
