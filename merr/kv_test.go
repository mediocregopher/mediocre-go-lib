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

	er := New("foo", "bar", "baz")
	kv := KV(er).KV()
	massert.Fatal(t, massert.Comment(
		massert.All(
			massert.Len(kv, 3),
			massert.Equal("foo", kv["err"]),
			massert.Equal("baz", kv["bar"]),
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
			massert.Len(kv, 4),
			massert.Equal("foo", kv["err"]),
			massert.Equal("baz", kv["bar"]),
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
			massert.Len(kv, 5),
			massert.Equal("foo", kv["err"]),
			massert.Equal("baz", kv["bar"]),
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
			massert.Len(kv, 6),
			massert.Equal("foo", kv["err"]),
			massert.Equal("baz", kv["bar"]),
			massert.Equal(true,
				strings.HasPrefix(kv["errSrc"].(string), "merr/kv_test.go:")),
			massert.Equal("1", kv["merr.A(k)"]),
			massert.Equal("2", kv["merr.B(k)"]),
			massert.Equal("3", kv["merr.C(k)"]),
		),
		"kv: %#v", kv,
	))

	er = WithKV(er, map[string]interface{}{"D": 4, "k": 5})
	kv = KV(er).KV()
	massert.Fatal(t, massert.Comment(
		massert.All(
			massert.Len(kv, 8),
			massert.Equal("foo", kv["err"]),
			massert.Equal("baz", kv["bar"]),
			massert.Equal(true,
				strings.HasPrefix(kv["errSrc"].(string), "merr/kv_test.go:")),
			massert.Equal("1", kv["merr.A(k)"]),
			massert.Equal("2", kv["merr.B(k)"]),
			massert.Equal("3", kv["merr.C(k)"]),
			massert.Equal(4, kv["D"]),
			massert.Equal(5, kv["merr.kvKey(k)"]),
		),
		"kv: %#v", kv,
	))
}
