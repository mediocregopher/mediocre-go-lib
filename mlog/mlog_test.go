package mlog

import (
	"bytes"
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
		massert.Nil(kv.KV()),
		massert.Len(kv.KV(), 0),
	))

	// test that the Set method returns a copy
	kv = KV{"foo": "a"}
	kv2 := kv.Set("bar", "wat")
	kv["bur"] = "ok"
	massert.Fatal(t, massert.All(
		massert.Equal(KV{"foo": "a", "bur": "ok"}, kv),
		massert.Equal(KV{"foo": "a", "bar": "wat"}, kv2),
	))
}

func TestLogger(t *T) {
	buf := new(bytes.Buffer)
	h := func(msg Message) error {
		return DefaultFormat(buf, msg)
	}

	l := NewLogger()
	l.SetHandler(h)
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
	l.Debug("foo")
	l.Info("bar")
	l.Warn("baz")
	l.Error("buz")
	massert.Fatal(t, massert.All(
		assertOut("~ INFO -- bar\n"),
		assertOut("~ WARN -- baz\n"),
		assertOut("~ ERROR -- buz\n"),
	))

	l.SetMaxLevel(WarnLevel)
	l.Debug("foo")
	l.Info("bar")
	l.Warn("baz")
	l.Error("buz", KV{"a": "b"})
	massert.Fatal(t, massert.All(
		assertOut("~ WARN -- baz\n"),
		assertOut("~ ERROR -- buz -- a=\"b\"\n"),
	))

	l2 := l.Clone()
	l2.SetMaxLevel(InfoLevel)
	l2.SetHandler(func(msg Message) error {
		msg.Description = String(strings.ToUpper(msg.Description.String()))
		return h(msg)
	})
	l2.Info("bar")
	l2.Warn("baz")
	l.Error("buz")
	massert.Fatal(t, massert.All(
		assertOut("~ INFO -- BAR\n"),
		assertOut("~ WARN -- BAZ\n"),
		assertOut("~ ERROR -- buz\n"),
	))

	l3 := l2.Clone()
	l3.SetKV(KV{"a": 1})
	l3.Info("foo", KV{"b": 2})
	l3.Info("bar", KV{"a": 2, "b": 3})
	massert.Fatal(t, massert.All(
		assertOut("~ INFO -- FOO -- a=\"1\" b=\"2\"\n"),
		assertOut("~ INFO -- BAR -- a=\"2\" b=\"3\"\n"),
	))

}

func TestDefaultFormat(t *T) {
	assertFormat := func(postfix string, msg Message) massert.Assertion {
		expectedRegex := regexp.MustCompile(`^~ ` + postfix + `\n$`)
		buf := bytes.NewBuffer(make([]byte, 0, 128))
		writeErr := DefaultFormat(buf, msg)
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

	msg := Message{Level: InfoLevel, Description: String("this is a test")}
	massert.Fatal(t, assertFormat("INFO -- this is a test", msg))

	msg.KVer = KV{}
	massert.Fatal(t, assertFormat("INFO -- this is a test", msg))

	msg.KVer = KV{"foo": "a"}
	massert.Fatal(t, assertFormat("INFO -- this is a test -- foo=\"a\"", msg))

	msg.KVer = KV{"foo": "a", "bar": "b"}
	massert.Fatal(t,
		assertFormat("INFO -- this is a test -- bar=\"b\" foo=\"a\"", msg))
}

func TestMerge(t *T) {
	assertMerge := func(exp KV, kvs ...KVer) massert.Assertion {
		return massert.Equal(exp.KV(), Merge(kvs...).KV())
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
			massert.Equal(map[string]interface{}{"a": "b"}, kv.KV()),
			massert.Equal(map[string]interface{}{"a": "b"}, mergedKV.KV()),
		))
	}
}

func TestPrefix(t *T) {
	kv := KV{"foo": "bar"}
	prefixKV := Prefix(kv, "aa")

	massert.Fatal(t, massert.All(
		massert.Equal(map[string]interface{}{"foo": "bar"}, kv.KV()),
		massert.Equal(map[string]interface{}{"aafoo": "bar"}, prefixKV.KV()),
		massert.Equal(map[string]interface{}{"foo": "bar"}, kv.KV()),
	))
}
