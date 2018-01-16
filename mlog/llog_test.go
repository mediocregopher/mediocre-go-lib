package mlog

import (
	"bytes"
	"io"
	"io/ioutil"
	"regexp"
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

	assert.Equal(t, KV{"foo": "a", "bar": "b"}, Merge(
		KV{"foo": "aaaaa"},
		KV{"foo": "a", "bar": "bbbbb"},
		KV{"bar": "b"},
	))
}

func TestLLog(t *T) {
	buf := new(bytes.Buffer)
	l := &Logger{
		WriteCloser: struct {
			io.Writer
			io.Closer
		}{
			Writer: buf,
			Closer: ioutil.NopCloser(nil),
		},
		testMsgWrittenCh: make(chan struct{}, 10),
	}

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

	l.SetMaxLevel(WarnLevel)
	l.Log(DebugLevel, "foo")
	l.Log(InfoLevel, "bar")
	l.Log(WarnLevel, "baz")
	l.Log(ErrorLevel, "buz", KV{"a": "b"})
	assertOut("~ WARN -- baz\n")
	assertOut("~ ERROR -- buz -- a=\"b\"\n")
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
