package jstream

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"sync"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/stretchr/testify/assert"
)

type cancelBuffer struct {
	lr *io.LimitedReader
}

func newCancelBuffer(b []byte) io.Reader {
	return &cancelBuffer{
		lr: &io.LimitedReader{
			R: bytes.NewBuffer(b),
			N: int64(len(b) / 2),
		},
	}
}

func (cb *cancelBuffer) Read(p []byte) (int, error) {
	if cb.lr.N == 0 {
		return 0, ErrCanceled
	}
	return cb.lr.Read(p)
}

func TestEncoderDecoder(t *T) {
	type testCase struct {
		typ    Type
		val    interface{}
		bytes  []byte
		stream []testCase
		cancel bool
	}

	var randTestCase func(Type, bool) testCase
	randTestCase = func(typ Type, cancelable bool) testCase {
		// if typ isn't given then use a random one
		if typ == "" {
			pick := mrand.Intn(5)
			switch {
			case pick == 0:
				typ = TypeStream
			case pick < 4:
				typ = TypeJSONValue
			default:
				typ = TypeByteBlob
			}
		}

		tc := testCase{
			typ:    typ,
			cancel: cancelable && mrand.Intn(10) == 0,
		}

		switch typ {
		case TypeJSONValue:
			tc.val = map[string]interface{}{
				mrand.Hex(8): mrand.Hex(8),
				mrand.Hex(8): mrand.Hex(8),
				mrand.Hex(8): mrand.Hex(8),
				mrand.Hex(8): mrand.Hex(8),
				mrand.Hex(8): mrand.Hex(8),
			}
			return tc
		case TypeByteBlob:
			tc.bytes = mrand.Bytes(mrand.Intn(256))
			return tc
		case TypeStream:
			for i := mrand.Intn(10); i > 0; i-- {
				tc.stream = append(tc.stream, randTestCase("", true))
			}
			return tc
		}
		panic("shouldn't get here")
	}

	tcLog := func(tcs ...testCase) []interface{} {
		return []interface{}{"%#v", tcs}
	}

	var assertRead func(*StreamReader, Element, testCase) bool
	assertRead = func(r *StreamReader, el Element, tc testCase) bool {
		l, success := tcLog(tc), true
		typ, err := el.Type()
		success = success && assert.NoError(t, err, l...)
		success = success && assert.Equal(t, tc.typ, typ, l...)

		switch typ {
		case TypeJSONValue:
			var val interface{}
			success = success && assert.NoError(t, el.Value(&val), l...)
			success = success && assert.Equal(t, tc.val, val, l...)
		case TypeByteBlob:
			br, err := el.Bytes()
			success = success && assert.NoError(t, err, l...)
			success = success && assert.Equal(t, uint(len(tc.bytes)), el.SizeHint(), l...)
			all, err := ioutil.ReadAll(br)
			if tc.cancel {
				success = success && assert.Equal(t, ErrCanceled, err, l...)
			} else {
				success = success && assert.NoError(t, err, l...)
				success = success && assert.Equal(t, tc.bytes, all, l...)
			}
		case TypeStream:
			innerR, err := el.Stream()
			success = success && assert.NoError(t, err, l...)
			success = success && assert.Equal(t, uint(len(tc.stream)), el.SizeHint(), l...)
			n := 0
			for {
				el := innerR.Next()
				if tc.cancel && el.Err == ErrCanceled {
					break
				} else if n == len(tc.stream) {
					success = success && assert.Equal(t, ErrStreamEnded, el.Err, l...)
					break
				}
				success = success && assertRead(innerR, el, tc.stream[n])
				n++
			}
		}
		return success
	}

	var assertWrite func(*StreamWriter, testCase) bool
	assertWrite = func(w *StreamWriter, tc testCase) bool {
		l, success := tcLog(tc), true
		switch tc.typ {
		case TypeJSONValue:
			success = success && assert.NoError(t, w.EncodeValue(tc.val), l...)
		case TypeByteBlob:
			if tc.cancel {
				r := newCancelBuffer(tc.bytes)
				err := w.EncodeBytes(uint(len(tc.bytes)), r)
				success = success && assert.Equal(t, ErrCanceled, err, l...)
			} else {
				r := bytes.NewBuffer(tc.bytes)
				err := w.EncodeBytes(uint(len(tc.bytes)), r)
				success = success && assert.NoError(t, err, l...)
			}
		case TypeStream:
			err := w.EncodeStream(uint(len(tc.stream)), func(innerW *StreamWriter) error {
				if len(tc.stream) == 0 && tc.cancel {
					return ErrCanceled
				}
				for i := range tc.stream {
					if tc.cancel && i == len(tc.stream)/2 {
						return ErrCanceled
					} else if !assertWrite(w, tc.stream[i]) {
						return errors.New("we got problems")
					}
				}
				return nil
			})
			if tc.cancel {
				success = success && assert.Equal(t, ErrCanceled, err, l...)
			} else {
				success = success && assert.NoError(t, err, l...)
			}
		}
		return success
	}

	do := func(tcs ...testCase) bool {
		// we keep a copy of all read/written bytes for debugging, but generally
		// don't actually log them
		ioR, ioW := io.Pipe()
		cpR, cpW := new(bytes.Buffer), new(bytes.Buffer)
		r, w := NewStreamReader(io.TeeReader(ioR, cpR)), NewStreamWriter(io.MultiWriter(ioW, cpW))

		readCh, writeCh := make(chan bool, 1), make(chan bool, 1)
		wg := new(sync.WaitGroup)
		wg.Add(2)
		go func() {
			success := true
			for _, tc := range tcs {
				success = success && assertRead(r, r.Next(), tc)
			}
			success = success && assert.Equal(t, io.EOF, r.Next().Err)
			readCh <- success
			ioR.Close()
			wg.Done()
		}()
		go func() {
			success := true
			for _, tc := range tcs {
				success = success && assertWrite(w, tc)
			}
			writeCh <- success
			ioW.Close()
			wg.Done()
		}()
		wg.Wait()

		//log.Printf("data written:%q", cpW.Bytes())
		//log.Printf("data read:   %q", cpR.Bytes())

		if !(<-readCh && <-writeCh) {
			assert.FailNow(t, "test case failed", tcLog(tcs...)...)
			return false
		}
		return true
	}

	// some basic test cases
	do() // empty stream
	do(randTestCase(TypeJSONValue, false))
	do(randTestCase(TypeByteBlob, false))
	do(
		randTestCase(TypeJSONValue, false),
		randTestCase(TypeJSONValue, false),
		randTestCase(TypeJSONValue, false),
	)
	do(
		randTestCase(TypeJSONValue, false),
		randTestCase(TypeByteBlob, false),
		randTestCase(TypeJSONValue, false),
	)
	do(
		randTestCase(TypeByteBlob, false),
		randTestCase(TypeByteBlob, false),
		randTestCase(TypeByteBlob, false),
	)
	do(
		randTestCase(TypeJSONValue, false),
		randTestCase(TypeStream, false),
		randTestCase(TypeJSONValue, false),
	)

	// some special cases, empty elements which are canceled
	do(testCase{typ: TypeStream, cancel: true})
	do(testCase{typ: TypeByteBlob, cancel: true})

	for i := 0; i < 1000; i++ {
		tc := randTestCase(TypeStream, false)
		do(tc.stream...)
	}
}
