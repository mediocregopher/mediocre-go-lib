package jstream

import (
	"encoding/json"
	"errors"
	"io"
)

type chanElement struct {
	Type     Type
	SizeHint uint
	Err      error
	Value    json.RawMessage
	Bytes    io.Reader
	Stream   chan chanElement
}

func chanWriter(sw *StreamWriter, ch <-chan chanElement) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		for el := range ch {
			var err error
			switch {
			case el.Err != nil:
				err = el.Err
			case el.Value != nil:
				err = sw.EncodeValue(el.Value)
			case el.Bytes != nil:
				err = sw.EncodeBytes(el.SizeHint, el.Bytes)
			case el.Stream != nil:
				err = sw.EncodeStream(el.SizeHint, func(innerSW *StreamWriter) error {
					// TODO this is a sticking point. The error which occurs in
					// here should be available to whatever is writing to
					// el.Stream, so it can know if it's wasting its time and
					// should be canceled
					innerErrCh := chanWriter(innerSW, el.Stream)
					return <-innerErrCh
				})
			default:
				err = errors.New("malformed chanElement, no fields set")
			}
			if err != nil {
				errCh <- err
				return
			}
		}
	}()
	return errCh
}

func chanReader(ch chan<- chanElement, sr *StreamReader) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		for {
			el := sr.Next()
			var err error
			switch {
			case el.Err == ErrStreamEnded:
				return
			case el.Err != nil:
				err = el.Err
			case el.Type == TypeValue:
				ch <- chanElement{
					Type:  TypeValue,
					Value: el.value,
				}
			case el.Type == TypeBytes:
				ch <- chanElement{
					Type:     TypeBytes,
					SizeHint: el.SizeHint,
					Bytes:    el.br,
				}
			case el.Type == TypeStream:
				// TODO this sucks for two reasons:
				// 1) the user can't set a buffer on this channel, like they
				//    could with the outermost one.
				// 2) the error returned from chanReader shouldn't get passed up
				//    if it's ErrCanceled, as that will cancel the outer channel
				//    too, which isn't necessary. But in that case there's no
				//    way to let the user know that the inner one was closed due
				//    to being canceled.
				innerCh := make(chan chanElement)
				ch <- chanElement{
					Type:     TypeStream,
					SizeHint: el.SizeHint,
					Stream:   innerCh,
				}
				innerSR, _ := el.DecodeStream() // already checked Err previously
				err = <-chanReader(innerCh, innerSR)
			}
		}
	}()
	return errCh
}
