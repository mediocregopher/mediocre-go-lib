package jstream

import (
	"bytes"
	"encoding/base64"
	"io"
)

type delimReader struct {
	r     io.Reader
	delim byte
	rest  []byte
}

func (dr *delimReader) Read(b []byte) (int, error) {
	if dr.delim != 0 {
		return 0, io.EOF
	}
	n, err := dr.r.Read(b)
	if i := bytes.IndexAny(b[:n], bbDelims); i >= 0 {
		dr.delim = b[i]
		dr.rest = append([]byte(nil), b[i+1:n]...)
		return i, err
	}
	return n, err
}

type bytesReader struct {
	dr  *delimReader
	dec io.Reader
}

func newBytesReader(r io.Reader) *bytesReader {
	dr := &delimReader{r: r}
	return &bytesReader{
		dr:  dr,
		dec: base64.NewDecoder(base64.StdEncoding, dr),
	}
}

func (br *bytesReader) Read(into []byte) (int, error) {
	n, err := br.dec.Read(into)
	if br.dr.delim == bbEnd {
		return n, io.EOF
	} else if br.dr.delim == bbCancel {
		return n, ErrCanceled
	}
	return n, err
}

// returns the bytes which were read off the underlying io.Reader but which
// haven't been consumed yet.
func (br *bytesReader) buffered() io.Reader {
	return bytes.NewBuffer(br.dr.rest)
}
