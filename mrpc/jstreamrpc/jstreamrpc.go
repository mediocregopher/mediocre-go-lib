// Package jstreamrpc implements the mrpc interfaces using the jstream protocol.
// This implementation makes a few design decisions which are not specified in
// the protocol, but which are good best practices none-the-less.
package jstreamrpc

import (
	"context"
	"errors"
	"io"

	"github.com/mediocregopher/mediocre-go-lib/jstream"
	"github.com/mediocregopher/mediocre-go-lib/mrpc"
)

// TODO Error?
// TODO SizeHints
// TODO it'd be nice if the types here played nice with mrpc.ReflectClient

type debug struct {
	Debug mrpc.Debug `json:"debug,omitempty"`
}

type reqHead struct {
	debug
	Method string `json:"method"`
}

type resTail struct {
	debug
	Error error `json:"err,omitempty"`
}

type ctxVal int

const (
	ctxValR ctxVal = iota
	ctxValW
)

func unmarshalBody(i interface{}, el jstream.Element) error {
	switch iT := i.(type) {
	case func(*jstream.StreamReader) error:
		stream, err := el.DecodeStream()
		if err != nil {
			return err
		}
		return iT(stream)
	case *io.Reader:
		ioR, err := el.DecodeBytes()
		if err != nil {
			return err
		}
		*iT = ioR
		return nil
	default:
		return el.DecodeValue(i)
	}
}

func marshalBody(w *jstream.StreamWriter, i interface{}) error {
	switch iT := i.(type) {
	case func(*jstream.StreamWriter) error:
		return w.EncodeStream(0, iT)
	case io.Reader:
		return w.EncodeBytes(0, iT)
	default:
		return w.EncodeValue(iT)
	}
}

// HandleCall TODO
//
// If this returns an error then both r and w should be discarded and no longer
// used.
func HandleCall(
	ctx context.Context,
	r *jstream.StreamReader,
	w *jstream.StreamWriter,
	h mrpc.Handler,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var head reqHead
	if err := r.Next().DecodeValue(&head); err != nil {
		return err
	} else if head.Method == "" {
		return errors.New("request head missing 'method' field")
	}

	var didReadBody bool
	ctx = context.WithValue(ctx, ctxValR, r)
	ctx = context.WithValue(ctx, ctxValW, w)

	rw := new(mrpc.ResponseWriter)
	h.ServeRPC(mrpc.Request{
		Context: ctx,
		Method:  head.Method,
		Unmarshal: func(i interface{}) error {
			didReadBody = true
			return unmarshalBody(i, r.Next())
		},
		Debug: head.debug.Debug,
	}, rw)

	// TODO unmarshaling request and marshaling response should be in
	// their own go-routines, just in case they are streams/bytes which depend
	// on each other

	resErr, resErrOk := rw.Response.(error)
	if resErrOk {
		if err := w.EncodeValue(nil); err != nil {
			return err
		}
	} else {
		if err := marshalBody(w, rw.Response); err != nil {
			return err
		}
	}

	// Reading the tail (and maybe discarding the body) should only be done once
	// marshalBody has finished
	if !didReadBody {
		if err := r.Next().Discard(); err != nil {
			return err
		}
	}

	if err := w.EncodeValue(resTail{
		debug: debug{Debug: rw.Debug},
		Error: resErr,
	}); err != nil {
		return err
	}

	return nil
}

/*
func sqr(r mrpc.Request, rw *mrpc.ResponseWriter) {
	ch := make(chan int)
	rw.Response = func(w *jstream.StreamWriter) error {
		for i := range ch {
			if err := w.EncodeValue(i * i); err != nil {
				return err
			}
		}
		return nil
	}

	go func() {
		defer close(ch)
		err := r.Unmarshal(func(r *jstream.StreamReader) error {
			for {
				var i int
				if err := r.Next().Value(&i); err == jstream.ErrStreamEnded {
					return nil
				} else if err != nil {
					return err
				}
				ch <- i
			}
		})
		if err != nil {
			panic("TODO")
		}
	}()
}
*/
