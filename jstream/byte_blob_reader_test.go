package jstream

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/stretchr/testify/assert"
)

type bbrTest struct {
	wsSuffix     []byte // whitespace
	body         []byte
	shouldCancel bool
	intoSize     int
}

func randBBRTest(minBodySize, maxBodySize int) bbrTest {
	var whitespace = []byte{' ', '\n', '\t', '\r'}
	genWhitespace := func(n int) []byte {
		ws := make([]byte, n)
		for i := range ws {
			ws[i] = whitespace[mtest.Rand.Intn(len(whitespace))]
		}
		return ws
	}

	body := mtest.RandBytes(minBodySize + mtest.Rand.Intn(maxBodySize-minBodySize))
	return bbrTest{
		wsSuffix: genWhitespace(mtest.Rand.Intn(10)),
		body:     body,
		intoSize: 1 + mtest.Rand.Intn(len(body)+1),
	}
}

func (bt bbrTest) msgAndArgs() []interface{} {
	return []interface{}{"bt:%#v len(body):%d", bt, len(bt.body)}
}

func (bt bbrTest) mkBytes() []byte {
	buf := new(bytes.Buffer)
	enc := base64.NewEncoder(base64.StdEncoding, buf)

	if bt.shouldCancel {
		enc.Write(bt.body[:len(bt.body)/2])
		enc.Close()
		buf.WriteByte(bbCancel)
	} else {
		enc.Write(bt.body)
		enc.Close()
		buf.WriteByte(bbEnd)
	}

	buf.Write(bt.wsSuffix)
	return buf.Bytes()
}

func (bt bbrTest) do(t *T) bool {
	buf := bytes.NewBuffer(bt.mkBytes())
	bbr := newByteBlobReader(buf)

	into := make([]byte, bt.intoSize)
	outBuf := new(bytes.Buffer)
	_, err := io.CopyBuffer(outBuf, bbr, into)
	if bt.shouldCancel {
		return assert.Equal(t, ErrCanceled, err, bt.msgAndArgs()...)
	}
	if !assert.NoError(t, err, bt.msgAndArgs()...) {
		return false
	}
	if !assert.Equal(t, bt.body, outBuf.Bytes(), bt.msgAndArgs()...) {
		return false
	}
	fullRest := append(bbr.dr.rest, buf.Bytes()...)
	if len(bt.wsSuffix) == 0 {
		return assert.Empty(t, fullRest, bt.msgAndArgs()...)
	}
	return assert.Equal(t, bt.wsSuffix, fullRest, bt.msgAndArgs()...)
}

func TestByteBlobReader(t *T) {
	// some sanity tests
	bbrTest{
		body:     []byte{2, 3, 4, 5},
		intoSize: 4,
	}.do(t)
	bbrTest{
		body:     []byte{2, 3, 4, 5},
		intoSize: 3,
	}.do(t)
	bbrTest{
		body:         []byte{2, 3, 4, 5},
		shouldCancel: true,
		intoSize:     3,
	}.do(t)

	// fuzz this bitch
	for i := 0; i < 50000; i++ {
		bt := randBBRTest(0, 1000)
		if !bt.do(t) {
			return
		}
		bt.shouldCancel = true
		if !bt.do(t) {
			return
		}
	}
}

func BenchmarkByteBlobReader(b *B) {
	type bench struct {
		bt    bbrTest
		body  []byte
		buf   *bytes.Reader
		cpBuf []byte
	}

	mkTestSet := func(minBodySize, maxBodySize int) []bench {
		n := 100
		benches := make([]bench, n)
		for i := range benches {
			bt := randBBRTest(minBodySize, maxBodySize)
			body := bt.mkBytes()
			benches[i] = bench{
				bt:    bt,
				body:  body,
				buf:   bytes.NewReader(nil),
				cpBuf: make([]byte, bt.intoSize),
			}
		}
		return benches
	}

	testRaw := func(b *B, benches []bench) {
		j := 0
		for i := 0; i < b.N; i++ {
			if j >= len(benches) {
				j = 0
			}
			benches[j].buf.Reset(benches[j].body)
			io.CopyBuffer(ioutil.Discard, benches[j].buf, benches[j].cpBuf)
			j++
		}
	}

	testBBR := func(b *B, benches []bench) {
		j := 0
		for i := 0; i < b.N; i++ {
			if j >= len(benches) {
				j = 0
			}
			benches[j].buf.Reset(benches[j].body)
			bbr := newByteBlobReader(benches[j].buf)
			io.CopyBuffer(ioutil.Discard, bbr, benches[j].cpBuf)
			j++
		}
	}

	benches := []struct {
		name                     string
		minBodySize, maxBodySize int
	}{
		{"small", 0, 1000},
		{"medium", 1000, 10000},
		{"large", 10000, 100000},
		{"xlarge", 100000, 1000000},
	}

	b.StopTimer()
	for i := range benches {
		b.Run(benches[i].name, func(b *B) {
			set := mkTestSet(benches[i].minBodySize, benches[i].maxBodySize)
			b.StartTimer()
			b.Run("raw", func(b *B) {
				testRaw(b, set)
			})
			b.Run("bbr", func(b *B) {
				testBBR(b, set)
			})
			b.StopTimer()
		})
	}
}
