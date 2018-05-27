package jstream

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"sync"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest"
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
		typ     Type
		val     interface{}
		bytes   []byte
		stream  []testCase
		cancel  bool
		discard bool
	}

	var randTestCase func(Type, bool) testCase
	randTestCase = func(typ Type, cancelable bool) testCase {
		// if typ isn't given then use a random one
		if typ == "" {
			pick := mtest.Rand.Intn(5)
			switch {
			case pick == 0:
				typ = TypeStream
			case pick < 4:
				typ = TypeValue
			default:
				typ = TypeBytes
			}
		}

		cancel := cancelable && mtest.Rand.Intn(10) == 0
		tc := testCase{
			typ:    typ,
			cancel: cancel,
			// using cancelable here is gross, but whatever
			discard: cancelable && !cancel && mtest.Rand.Intn(20) == 0,
		}

		switch typ {
		case TypeValue:
			tc.val = map[string]interface{}{
				mtest.RandHex(8): mtest.RandHex(8),
				mtest.RandHex(8): mtest.RandHex(8),
				mtest.RandHex(8): mtest.RandHex(8),
				mtest.RandHex(8): mtest.RandHex(8),
				mtest.RandHex(8): mtest.RandHex(8),
			}
			return tc
		case TypeBytes:
			tc.bytes = mtest.RandBytes(mtest.Rand.Intn(256))
			return tc
		case TypeStream:
			for i := mtest.Rand.Intn(10); i > 0; i-- {
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
		success = success && assert.Equal(t, tc.typ, el.Type, l...)

		switch el.Type {
		case TypeValue:
			if tc.discard {
				success = success && assert.NoError(t, el.Discard())
				break
			}

			var val interface{}
			success = success && assert.NoError(t, el.DecodeValue(&val), l...)
			success = success && assert.Equal(t, tc.val, val, l...)
		case TypeBytes:
			br, err := el.DecodeBytes()
			success = success && assert.NoError(t, err, l...)
			success = success && assert.Equal(t, uint(len(tc.bytes)), el.SizeHint, l...)

			// if we're discarding we read some of the bytes and then will
			// discard the rest
			var discardKeep int
			if tc.discard {
				discardKeep = mtest.Rand.Intn(len(tc.bytes) + 1)
				br = io.LimitReader(br, int64(discardKeep))
			}

			all, err := ioutil.ReadAll(br)
			if tc.cancel {
				success = success && assert.Equal(t, ErrCanceled, err, l...)
			} else if tc.discard {
				success = success && assert.NoError(t, err, l...)
				success = success && assert.Equal(t, tc.bytes[:discardKeep], all, l...)
				success = success && assert.NoError(t, el.Discard())

			} else {
				success = success && assert.NoError(t, err, l...)
				success = success && assert.Equal(t, tc.bytes, all, l...)
			}
		case TypeStream:
			innerR, err := el.DecodeStream()
			success = success && assert.NoError(t, err, l...)
			success = success && assert.Equal(t, uint(len(tc.stream)), el.SizeHint, l...)

			// if we're discarding we read some of the elements and then will
			// discard the rest
			var discardKeep int
			if tc.discard {
				discardKeep = mtest.Rand.Intn(len(tc.stream) + 1)
			}

			n := 0
			for {
				if tc.discard && n == discardKeep {
					break
				}

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
			if tc.discard {
				success = success && assert.NoError(t, el.Discard())
			}
		}
		return success
	}

	var assertWrite func(*StreamWriter, testCase) bool
	assertWrite = func(w *StreamWriter, tc testCase) bool {
		l, success := tcLog(tc), true
		switch tc.typ {
		case TypeValue:
			success = success && assert.NoError(t, w.EncodeValue(tc.val), l...)
		case TypeBytes:
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
	do(randTestCase(TypeValue, false))
	do(randTestCase(TypeBytes, false))
	do(
		randTestCase(TypeValue, false),
		randTestCase(TypeValue, false),
		randTestCase(TypeValue, false),
	)
	do(
		randTestCase(TypeValue, false),
		randTestCase(TypeBytes, false),
		randTestCase(TypeValue, false),
	)
	do(
		randTestCase(TypeBytes, false),
		randTestCase(TypeBytes, false),
		randTestCase(TypeBytes, false),
	)
	do(
		randTestCase(TypeValue, false),
		randTestCase(TypeStream, false),
		randTestCase(TypeValue, false),
	)

	// some special cases, empty elements which are canceled
	do(testCase{typ: TypeStream, cancel: true})
	do(testCase{typ: TypeBytes, cancel: true})

	for i := 0; i < 1000; i++ {
		tc := randTestCase(TypeStream, false)
		do(tc.stream...)
	}
}

func TestDoubleDiscardBytes(t *T) {
	buf := new(bytes.Buffer)
	w := NewStreamWriter(buf)
	b1, b2 := mtest.RandBytes(10), mtest.RandBytes(10)
	assert.NoError(t, w.EncodeBytes(uint(len(b1)), bytes.NewBuffer(b1)))
	assert.NoError(t, w.EncodeBytes(uint(len(b2)), bytes.NewBuffer(b2)))

	r := NewStreamReader(buf)
	el1 := r.Next()
	r1, err := el1.DecodeBytes()
	assert.NoError(t, err)

	b1got, err := ioutil.ReadAll(r1)
	assert.NoError(t, err)
	assert.Equal(t, b1, b1got)

	// discarding an already consumed bytes shouldn't do anything
	assert.NoError(t, el1.Discard())

	r2, err := r.Next().DecodeBytes()
	assert.NoError(t, err)
	b2got, err := ioutil.ReadAll(r2)
	assert.NoError(t, err)
	assert.Equal(t, b2, b2got)
}
