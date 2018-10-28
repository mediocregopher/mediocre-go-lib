package mlog

import (
	"bytes"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestTruncate(t *T) {
	massert.Fatal(t, massert.All(
		massert.Equal("abc", Truncate("abc", 4)),
		massert.Equal("abc", Truncate("abc", 3)),
		massert.Equal("ab...", Truncate("abc", 2)),
	))
}

func TestKV(t *T) {
	var kv KV
	massert.Fatal(t, massert.All(
		massert.Not(massert.Nil(kv.KV())),
		massert.Len(kv.KV(), 0),
	))

	// test that the KV method returns a new KV instance
	kv = KV{"foo": "a"}
	kv2 := kv.KV()
	kv["bur"] = "b"
	kv2["bar"] = "bb"
	massert.Fatal(t, massert.All(
		massert.Equal(KV{"foo": "a", "bur": "b"}, kv),
		massert.Equal(KV{"foo": "a", "bar": "bb"}, kv2),
	))

	// test that the Set method returns a new KV instance
	kv = KV{"foo": "a"}
	kv2 = kv.Set("bar", "wat")
	kv["bur"] = "ok"
	massert.Fatal(t, massert.All(
		massert.Equal(KV{"foo": "a", "bur": "ok"}, kv),
		massert.Equal(KV{"foo": "a", "bar": "wat"}, kv2),
	))
}

func TestLogger(t *T) {
	buf := new(bytes.Buffer)
	l := NewLogger(struct {
		io.Writer
		io.Closer
	}{
		Writer: buf,
		Closer: ioutil.NopCloser(nil),
	})
	l.testMsgWrittenCh = make(chan struct{}, 10)

	assertOut := func(expected string) massert.Assertion {
		select {
		case <-l.testMsgWrittenCh:
		case <-time.After(1 * time.Second):
			return massert.Errf("waited too long for msg to write")
		}
		out, err := buf.ReadString('\n')
		return massert.All(
			massert.Nil(err),
			massert.Equal(expected, out),
		)
	}

	// Default max level should be INFO
	l.Log(DebugLevel, "foo")
	l.Log(InfoLevel, "bar")
	l.Log(WarnLevel, "baz")
	l.Log(ErrorLevel, "buz")
	massert.Fatal(t, massert.All(
		assertOut("~ INFO -- bar\n"),
		assertOut("~ WARN -- baz\n"),
		assertOut("~ ERROR -- buz\n"),
	))

	l.SetMaxLevel(WarnLevel)
	l.Log(DebugLevel, "foo")
	l.Log(InfoLevel, "bar")
	l.Log(WarnLevel, "baz")
	l.Log(ErrorLevel, "buz", KV{"a": "b"})
	massert.Fatal(t, massert.All(
		assertOut("~ WARN -- baz\n"),
		assertOut("~ ERROR -- buz -- a=\"b\"\n"),
	))

	l2 := l.Clone()
	l2.SetMaxLevel(InfoLevel)
	l2.SetWriteFn(func(w io.Writer, msg Message) error {
		msg.Msg = strings.ToUpper(msg.Msg)
		return DefaultWriteFn(w, msg)
	})
	l2.Log(InfoLevel, "bar")
	l2.Log(WarnLevel, "baz")
	l.Log(ErrorLevel, "buz")
	massert.Fatal(t, massert.All(
		assertOut("~ INFO -- BAR\n"),
		assertOut("~ WARN -- BAZ\n"),
		assertOut("~ ERROR -- buz\n"),
	))
}

func TestDefaultWriteFn(t *T) {
	assertFormat := func(postfix string, msg Message) massert.Assertion {
		expectedRegex := regexp.MustCompile(`^~ ` + postfix + `\n$`)
		buf := bytes.NewBuffer(make([]byte, 0, 128))
		writeErr := DefaultWriteFn(buf, msg)
		line, err := buf.ReadString('\n')
		return massert.Comment(
			massert.All(
				massert.Nil(writeErr),
				massert.Nil(err),
				massert.Equal(true, expectedRegex.MatchString(line)),
			),
			"line:%q", line,
		)
	}

	msg := Message{Level: InfoLevel, Msg: "this is a test"}
	massert.Fatal(t, assertFormat("INFO -- this is a test", msg))

	msg.KV = KV{}.KV()
	massert.Fatal(t, assertFormat("INFO -- this is a test", msg))

	msg.KV = KV{"foo": "a"}.KV()
	massert.Fatal(t, assertFormat("INFO -- this is a test -- foo=\"a\"", msg))

	msg.KV = KV{"foo": "a", "bar": "b"}.KV()
	massert.Fatal(t,
		assertFormat("INFO -- this is a test -- bar=\"b\" foo=\"a\"", msg))
}

func TestMerge(t *T) {
	assertMerge := func(exp KV, kvs ...KVer) massert.Assertion {
		return massert.Equal(exp, Merge(kvs...).KV())
	}

	massert.Fatal(t, massert.All(
		assertMerge(KV{}),
		assertMerge(KV{}, nil),
		assertMerge(KV{}, nil, nil),

		assertMerge(KV{"a": "a"}, KV{"a": "a"}),
		assertMerge(KV{"a": "a"}, nil, KV{"a": "a"}),
		assertMerge(KV{"a": "a"}, KV{"a": "a"}, nil),

		assertMerge(
			KV{"a": "a", "b": "b"},
			KV{"a": "a"}, KV{"b": "b"},
		),
		assertMerge(
			KV{"a": "a", "b": "b"},
			KV{"a": "a"}, KV{"b": "b"},
		),
		assertMerge(
			KV{"a": "b"},
			KV{"a": "a"}, KV{"a": "b"},
		),
	))

	// Merge should _not_ call KV() on the inner KVers until the outer one is
	// called.
	{
		kv := KV{"a": "a"}
		mergedKV := Merge(kv)
		kv["a"] = "b"
		massert.Fatal(t, massert.All(
			massert.Equal(KV{"a": "b"}, kv),
			massert.Equal(KV{"a": "b"}, kv.KV()),
			massert.Equal(KV{"a": "b"}, mergedKV.KV()),
		))
	}
}

func TestPrefix(t *T) {
	kv := KV{"foo": "bar"}
	prefixKV := Prefix(kv, "aa")

	massert.Fatal(t, massert.All(
		massert.Equal(kv.KV(), KV{"foo": "bar"}),
		massert.Equal(prefixKV.KV(), KV{"aafoo": "bar"}),
		massert.Equal(kv.KV(), KV{"foo": "bar"}),
	))
}
