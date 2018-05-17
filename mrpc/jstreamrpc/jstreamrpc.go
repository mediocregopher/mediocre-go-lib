// Package jstreamrpc implements the mrpc interfaces using the jstream protocol.
// This implementation makes a few design decisions which are not specified in
// the protocol, but which are good best practices none-the-less.
package jstreamrpc

import (
	"context"
	"errors"
	"net"

	"github.com/mediocregopher/mediocre-go-lib/jstream"
	"github.com/mediocregopher/mediocre-go-lib/mrpc"
)

// TODO Error?
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
	case *jstream.Element:
		*iT = el
		return nil
	default:
		return el.DecodeValue(i)
	}
}

func marshalBody(w *jstream.StreamWriter, i interface{}) error {
	switch iT := i.(type) {
	case func(*jstream.StreamWriter) error:
		return iT(w)
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

	ctx = context.WithValue(ctx, ctxValR, r)
	ctx = context.WithValue(ctx, ctxValW, w)

	body := r.Next()
	if body.Err != nil {
		return body.Err
	}

	rw := new(mrpc.ResponseWriter)
	h.ServeRPC(mrpc.Request{
		Context: ctx,
		Method:  head.Method,
		Unmarshal: func(i interface{}) error {
			return unmarshalBody(i, body)
		},
		Debug: head.debug.Debug,
	}, rw)

	resErr, resErrOk := rw.Response.(error)
	if resErrOk {
		if err := w.EncodeValue(nil); err != nil {
			return err
		}
	} else {
		if err := marshalBody(w, rw.Response); err != nil {
			if _, ok := err.(net.Error); ok {
				return err
			}
			resErr = err
		}
	}

	if err := w.EncodeValue(resTail{
		debug: debug{Debug: rw.Debug},
		Error: resErr,
	}); err != nil {
		return err
	}

	// make sure the body has been consumed before returning
	if err := body.Discard(); err != nil {
		return err
	}

	return nil
}

/*
func sqr(r mrpc.Request, rw *mrpc.ResponseWriter) {
	var el jstream.Element
	if err := r.Unmarshal(&el); err != nil {
		rw.Response = err
		return
	}

	sr, err := el.DecodeStream()
	if err != nil {
		rw.Response = err
		return
	}

	ch := make(chan int)
	go func() {
		defer close(ch)
		for {
			var i int
			if err := sr.Next().Value(&i); err == jstream.ErrStreamEnded {
				return
			} else if err != nil {
				panic("TODO")
			}
			ch <- i
		}
	}()

	rw.Response = func(sw *jstream.StreamWriter) error {
		return sw.EncodeStream(0, func(sw *jstream.StreamWriter) error {
			for i := range ch {
				if err := sw.EncodeValue(i * i); err != nil {
					return err
				}
			}
			return nil
		})
	}
}
*/
