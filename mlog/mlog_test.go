package mlog

import (
	"bytes"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	. "testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncate(t *T) {
	assert.Equal(t, "abc", Truncate("abc", 4))
	assert.Equal(t, "abc", Truncate("abc", 3))
	assert.Equal(t, "ab...", Truncate("abc", 2))
}

func TestKV(t *T) {
	var kv KV
	assert.NotNil(t, kv.KV())
	assert.Empty(t, kv.KV())

	kv = KV{"foo": "a"}
	kv2 := KV(kv.KV())
	kv["bar"] = "b"
	kv2["bar"] = "bb"
	assert.Equal(t, KV{"foo": "a", "bar": "b"}, kv)
	assert.Equal(t, KV{"foo": "a", "bar": "bb"}, kv2)

	kv = KV{"foo": "a"}
	kv2 = kv.Set("bar", "wat")
	assert.Equal(t, KV{"foo": "a"}, kv)
	assert.Equal(t, KV{"foo": "a", "bar": "wat"}, kv2)
}

func TestLLog(t *T) {
	buf := new(bytes.Buffer)
	l := NewLogger(struct {
		io.Writer
		io.Closer
	}{
		Writer: buf,
		Closer: ioutil.NopCloser(nil),
	})
	l.testMsgWrittenCh = make(chan struct{}, 10)

	assertOut := func(expected string) {
		select {
		case <-l.testMsgWrittenCh:
		case <-time.After(1 * time.Second):
			t.Fatal("waited too long for msg to write")
		}
		out, err := buf.ReadString('\n')
		require.NoError(t, err)
		assert.Equal(t, expected, out)
	}

	// Default max level should be INFO
	l.Log(DebugLevel, "foo")
	l.Log(InfoLevel, "bar")
	l.Log(WarnLevel, "baz")
	l.Log(ErrorLevel, "buz")
	assertOut("~ INFO -- bar\n")
	assertOut("~ WARN -- baz\n")
	assertOut("~ ERROR -- buz\n")

	{
		l := l.WithMaxLevel(WarnLevel)
		l.Log(DebugLevel, "foo")
		l.Log(InfoLevel, "bar")
		l.Log(WarnLevel, "baz")
		l.Log(ErrorLevel, "buz", KV{"a": "b"})
		assertOut("~ WARN -- baz\n")
		assertOut("~ ERROR -- buz -- a=\"b\"\n")
	}

	{
		l2 := l.WithWriteFn(func(w io.Writer, msg Message) error {
			msg.Msg = strings.ToUpper(msg.Msg)
			return DefaultWriteFn(w, msg)
		})
		l2.Log(InfoLevel, "bar")
		l2.Log(WarnLevel, "baz")
		l.Log(ErrorLevel, "buz")
		assertOut("~ INFO -- BAR\n")
		assertOut("~ WARN -- BAZ\n")
		assertOut("~ ERROR -- buz\n")
	}
}

func TestDefaultWriteFn(t *T) {
	assertFormat := func(postfix string, msg Message) {
		expectedRegex := regexp.MustCompile(`^~ ` + postfix + `\n$`)
		buf := bytes.NewBuffer(make([]byte, 0, 128))
		assert.NoError(t, DefaultWriteFn(buf, msg))
		line, err := buf.ReadString('\n')
		require.NoError(t, err)
		assert.True(t, expectedRegex.MatchString(line), "regex: %q line: %q", expectedRegex.String(), line)
	}

	msg := Message{Level: InfoLevel, Msg: "this is a test"}
	assertFormat("INFO -- this is a test", msg)

	msg.KV = KV{}.KV()
	assertFormat("INFO -- this is a test", msg)

	msg.KV = KV{"foo": "a"}.KV()
	assertFormat("INFO -- this is a test -- foo=\"a\"", msg)

	msg.KV = KV{"foo": "a", "bar": "b"}.KV()
	assertFormat("INFO -- this is a test -- bar=\"b\" foo=\"a\"", msg)
}

func TestMerge(t *T) {
	assertMerge := func(exp KV, kvs ...KVer) {
		got := merge(kvs...)
		assert.Equal(t, exp, got)
	}

	assertMerge(KV{})
	assertMerge(KV{}, nil)
	assertMerge(KV{}, nil, nil)

	assertMerge(KV{"a": "a"}, KV{"a": "a"})
	assertMerge(KV{"a": "a"}, nil, KV{"a": "a"})
	assertMerge(KV{"a": "a"}, KV{"a": "a"}, nil)

	assertMerge(
		KV{"a": "a", "b": "b"},
		KV{"a": "a"}, KV{"b": "b"},
	)
	assertMerge(
		KV{"a": "a", "b": "b"},
		KV{"a": "a"}, KV{"b": "b"},
	)
	assertMerge(
		KV{"a": "b"},
		KV{"a": "a"}, KV{"a": "b"},
	)
}
