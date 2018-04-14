// Package jstream defines and implements the JSON Stream protocol
//
// Purpose
//
// The purpose of the jstream protocol is to provide a very simple layer on top
// of an existing JSON implementation to allow for streaming arbitrary numbers
// of JSON objects and byte blobs of arbitrary size in a standard way, and to
// allow for embedding streams within each other.
//
// The order of priorities when designing jstream is as follows:
//	1) Protocol simplicity
//	2) Implementation simplicity
//	3) Efficiency, both in parsing speed and bandwidth
//
// The justification for this is that protocol simplicity generally spills into
// implementation simplicity anyway, and accounts for future languages which
// have different properties than current ones. Parsing speed isn't much of a
// concern when reading data off a network (the primary use-case here), as RTT
// is always going to be the main blocker. Bandwidth can be a concern, but it's
// one better solved by wrapping the byte stream with a compressor.
//
// jstream protocol
//
// The jstream protocol is carried over a byte stream (in go: an io.Reader). To
// read the protocol a JSON object is read off the byte stream and inspected to
// determine what kind of jstream element it is.
//
// Multiple jstream elements are sequentially read off the same byte stream.
// Each element may be separated from the other by any amount of whitespace,
// with whitespace being defined as spaces, tabs, carriage returns, and/or
// newlines.
//
// jstream elements
//
// There are three jstream element types:
//
// * JSON Value: Any JSON value
// * Byte Blob: A stream of bytes of unknown, and possibly infinite, size
// * Stream: A heterogenous sequence of jstream elements of unknown, and
//   possibly infinite, size
//
// JSON Value elements are defined as being JSON objects with a `val` field. The
// value of that field is the JSON Value.
//
//	{ "val":{"foo":"bar"} }
//
// Byte Blob elements are defined as being a JSON object with a `bytesStart`
// field with a value of `true`. Immediately following the JSON object are the
// bytes which are the Byte Blob, encoded using standard base64. Immediately
// following the encoded bytes is the character `$`, to indicate the bytes have
// been completely written. Alternatively the character `!` may be written
// immediately after the bytes to indicate writing was canceled prematurely by
// the writer.
//
//	{ "bytesStart":true }wXnxQHgUO8g=$
//	{ "bytesStart":true }WGYcTI8=!
//
// The JSON object may also contain a `sizeHint` field, which gives the
// estimated number of bytes in the Byte Blob (excluding the trailing
// delimiter). The hint is neither required to exist or be accurate if it does.
// The trailing delimeter (`$` or `!`) is required to be sent even if the hint
// is sent.
//
// Stream elements are defined as being a JSON object with a `streamStart` field
// with a value of `true`. Immediately following the JSON object will be zero
// more jstream elements of any type, possibly separated by whitespace. Finally
// the Stream is ended with another JSON object with a `streamEnd` field with a
// value of `true`.
//
//	{ "streamStart":true }
//		{ "val":{"foo":"bar"} }
//		{ "bytesStart":true }7TdlDQOnA6isxD9C$
//	{ "streamEnd":true }
//
// A Stream may also be prematurely canceled by the sending of a JSON object
// with the `streamCancel` field set to `true` (in place of one with `streamEnd`
// set to `true`).
//
// The Stream's original JSON object (the "head") may also have a `sizeHint`
// field, which gives the estimated number of jstream elements in the Stream.
// The hint is neither required to exist or be accurate if it does. The tail
// JSON object (with the `streamEnd` field) is required even if `sizeHint` is
// given.
//
// One of the elements in a Stream may itself be a Stream. In this way Streams
// may be embedded within each other.
//
// Here's an example of a complex Stream, which carries within it two different
// streams and some other elements:
//
//	{ "streamStart":true }
//		{ "val":{"foo":"bar" }
//		{ "streamStart":true, "sizeHint":2 }
//			{ "val":{"foo":"baz"} }
//			{ "val":{"foo":"biz"} }
//		{ "streamEnd":true }
//		{ "bytesStart":true }X7KCpLIjqIBJt9vA$
//		{ "streamStart":true }
//			{ "bytesStart":true }0jT+kNCuxHywUYy0$
//			{ "bytesStart":true }LUqjR6OACB2p1BG4$
//		{ "streamEnd":true }
//	{ "streamEnd":true }
//
// Finally, the byte stream off of which the jstream is based (i.e. the
// io.Reader) is implicitly treated as a Stream, with the Stream ending when the
// byte stream is closed.
//
package jstream

// TODO figure out how to expose the json.Encoder/Decoders so that users can set
// custom options on them (like UseNumber and whatnot)

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

// byte blob constants
const (
	bbEnd    = '$'
	bbCancel = '!'
	bbDelims = string(bbEnd) + string(bbCancel)
)

// Type is used to enumerate the types of jstream elements
type Type string

// The jstream element types
const (
	TypeJSONValue Type = "jsonValue"
	TypeByteBlob  Type = "byteBlob"
	TypeStream    Type = "stream"
)

// ErrWrongType is an error returned by the Decode* methods on Decoder when the
// wrong decoding method has been called for the element which was read. The
// error contains the actual type of the element.
type ErrWrongType struct {
	Actual Type
}

func (err ErrWrongType) Error() string {
	return fmt.Sprintf("wrong type, actual type is %q", err.Actual)
}

var (
	// ErrCanceled is returned when reading either a Byte Blob or a Stream,
	// indicating that the writer has prematurely canceled the element.
	ErrCanceled = errors.New("canceled by writer")

	// ErrStreamEnded is returned from Next when the Stream being read has been
	// ended by the writer.
	ErrStreamEnded = errors.New("stream ended")
)

type element struct {
	Value json.RawMessage `json:"val,omitempty"`

	BytesStart bool `json:"bytesStart,omitempty"`

	StreamStart  bool `json:"streamStart,omitempty"`
	StreamEnd    bool `json:"streamEnd,omitempty"`
	StreamCancel bool `json:"streamCancel,omitempty"`

	SizeHint uint `json:"sizeHint,omitempty"`
}

// Element is a single jstream element which is read off a StreamReader.
//
// If a method is called which expects a particular Element type (e.g.
// DecodeValue, which expects a JSONValue Element) but the Element is not of
// that type then an ErrWrongType will be returned.
//
// If there was an error reading the Element off the StreamReader that error is
// kept in the Element and returned from any method call.
type Element struct {
	element

	// Err will be set if the StreamReader encountered an error while reading
	// the next Element. If set then the Element is otherwise unusable.
	//
	// Err may be ErrCanceled or ErrStreamEnded, which would indicate the end of
	// the stream but would not indicate the StreamReader is no longer usable,
	// depending on the behavior of the writer on the other end.
	Err error

	// needed for byte blobs and streams
	sr *StreamReader
}

// Type returns the Element's Type, or an error
func (el Element) Type() (Type, error) {
	if el.Err != nil {
		return "", el.Err
	} else if el.element.StreamStart {
		return TypeStream, nil
	} else if el.element.BytesStart {
		return TypeByteBlob, nil
	} else if len(el.element.Value) > 0 {
		return TypeJSONValue, nil
	}
	return "", errors.New("malformed Element, can't determine type")
}

func (el Element) assertType(is Type) error {
	typ, err := el.Type()
	if err != nil {
		return err
	} else if typ != is {
		return ErrWrongType{Actual: typ}
	}
	return nil
}

// Value attempts to unmarshal a JSON Value Element's value into the given
// receiver.
//
// This method should not be called more than once.
func (el Element) Value(i interface{}) error {
	if err := el.assertType(TypeJSONValue); err != nil {
		return err
	}
	return json.Unmarshal(el.element.Value, i)
}

// SizeHint returns the size hint which may have been optionally sent for
// ByteBlob and Stream elements, or zero. The hint is never required to be
// sent or to be accurate.
func (el Element) SizeHint() uint {
	return el.element.SizeHint
}

// Bytes returns an io.Reader which will contain the contents of a ByteBlob
// element. The io.Reader _must_ be read till io.EOF or ErrCanceled before the
// StreamReader may be used again.
//
// This method should not be called more than once.
func (el Element) Bytes() (io.Reader, error) {
	if err := el.assertType(TypeByteBlob); err != nil {
		return nil, err
	}
	return el.sr.readBytes(), nil
}

// Stream returns the embedded stream represented by this Element as a
// StreamReader. The returned StreamReader _must_ be iterated (via the Next
// method) till ErrStreamEnded or ErrCanceled is returned before the original
// StreamReader may be used again.
//
// This method should not be called more than once.
func (el Element) Stream() (*StreamReader, error) {
	if err := el.assertType(TypeStream); err != nil {
		return nil, err
	}
	return el.sr, nil
}

// Discard reads whatever of this Element's data may be left on the StreamReader
// it came from and discards it, making the StreamReader ready to have Next call
// on it again.
//
// If the Element is a Byte Blob and is ended with io.EOF, or if the Element is
// a Stream and is ended with ErrStreamEnded then this returns nil. If either is
// canceled this also returns nil. All other errors are returned.
func (el Element) Discard() error {
	typ, err := el.Type()
	if err != nil {
		return err
	}
	switch typ {
	case TypeByteBlob:
		r, _ := el.Bytes()
		_, err := io.Copy(ioutil.Discard, r)
		if err == ErrCanceled {
			return nil
		}
		return err
	case TypeStream:
		stream, _ := el.Stream()
		for {
			nextEl := stream.Next()
			if nextEl.Err == ErrStreamEnded || nextEl.Err == ErrCanceled {
				return nil
			} else if nextEl.Err != nil {
				return nextEl.Err
			} else if err := nextEl.Discard(); err != nil {
				return err
			}
		}
	default: // TypeJSONValue
		return nil
	}
}

// StreamReader represents a Stream from which Elements may be read using the
// Next method.
type StreamReader struct {
	orig io.Reader

	// only one of these can be set at a time
	dec *json.Decoder
	bbr *byteBlobReader
}

// NewStreamReader takes an io.Reader and interprets it as a jstream Stream.
func NewStreamReader(r io.Reader) *StreamReader {
	return &StreamReader{orig: r}
}

// pulls buffered bytes out of either the json.Decoder or byteBlobReader, if
// possible, and returns an io.MultiReader of those and orig. Will also set the
// json.Decoder/byteBlobReader to nil if that's where the bytes came from.
func (sr *StreamReader) multiReader() io.Reader {
	if sr.dec != nil {
		buf := sr.dec.Buffered()
		sr.dec = nil
		return io.MultiReader(buf, sr.orig)
	} else if sr.bbr != nil {
		buf := sr.bbr.buffered()
		sr.bbr = nil
		return io.MultiReader(buf, sr.orig)
	}
	return sr.orig
}

// Next reads, decodes, and returns the next Element off the StreamReader. If
// the Element is a ByteBlob or embedded Stream then it _must_ be fully consumed
// before Next is called on this StreamReader again.
//
// The returned Element's Err field will be ErrStreamEnd if the Stream was
// ended, or ErrCanceled if it was canceled, and this StreamReader should not be
// used again in those cases.
//
// If the underlying io.Reader is closed the returned Err field will be io.EOF.
func (sr *StreamReader) Next() Element {
	if sr.dec == nil {
		sr.dec = json.NewDecoder(sr.multiReader())
	}

	var el element
	var err error
	if err = sr.dec.Decode(&el); err != nil {
		// welp
	} else if el.StreamEnd {
		err = ErrStreamEnded
	} else if el.StreamCancel {
		err = ErrCanceled
	}
	if err != nil {
		return Element{Err: err}
	}
	return Element{sr: sr, element: el}
}

func (sr *StreamReader) readBytes() *byteBlobReader {
	sr.bbr = newByteBlobReader(sr.multiReader())
	return sr.bbr
}

////////////////////////////////////////////////////////////////////////////////

// StreamWriter represents a Stream to which Elements may be written using any
// of the Encode methods.
type StreamWriter struct {
	w   io.Writer
	enc *json.Encoder
}

// NewStreamWriter takes an io.Writer and returns a StreamWriter which will
// write to it.
func NewStreamWriter(w io.Writer) *StreamWriter {
	return &StreamWriter{w: w, enc: json.NewEncoder(w)}
}

// EncodeValue marshals the given value and writes it to the Stream as a
// JSONValue element.
func (sw *StreamWriter) EncodeValue(i interface{}) error {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	return sw.enc.Encode(element{Value: b})
}

// EncodeBytes copies the given io.Reader, until io.EOF, onto the Stream as a
// ByteBlob element. This method will block until copying is completed or an
// error is encountered.
//
// If the io.Reader returns any error which isn't io.EOF then the ByteBlob is
// canceled and that error is returned from this method. Otherwise nil is
// returned.
//
// sizeHint may be given if it's known or can be guessed how many bytes the
// io.Reader will read out.
func (sw *StreamWriter) EncodeBytes(sizeHint uint, r io.Reader) error {
	if err := sw.enc.Encode(element{
		BytesStart: true,
		SizeHint:   sizeHint,
	}); err != nil {
		return err

	}

	enc := base64.NewEncoder(base64.StdEncoding, sw.w)
	if _, err := io.Copy(enc, r); err != nil {
		// if canceling doesn't work then the whole connection is broken and
		// it's not worth doing anything about. if nothing else the brokeness of
		// it will be discovered the next time it is used.
		sw.w.Write([]byte{bbCancel})
		return err
	} else if err := enc.Close(); err != nil {
		return err
	} else if _, err := sw.w.Write([]byte{bbEnd}); err != nil {
		return err
	}

	return nil
}

// EncodeStream encodes an embedded Stream element onto the Stream. The callback
// is given a new StreamWriter which represents the embedded Stream and to which
// any elemens may be written. This methods blocks until the callback has
// returned.
//
// If the callback returns nil the Stream is ended normally. If it returns
// anything else the embedded Stream is canceled and that error is returned from
// this method.
//
// sizeHint may be given if it's known or can be guessed how many elements will
// be in the embedded Stream.
func (sw *StreamWriter) EncodeStream(sizeHint uint, fn func(*StreamWriter) error) error {
	if err := sw.enc.Encode(element{
		StreamStart: true,
		SizeHint:    sizeHint,
	}); err != nil {
		return err

	} else if err := fn(sw); err != nil {
		// as when canceling a byte blob, we don't really care if this errors
		sw.enc.Encode(element{StreamCancel: true})
		return err
	}
	return sw.enc.Encode(element{StreamEnd: true})
}
