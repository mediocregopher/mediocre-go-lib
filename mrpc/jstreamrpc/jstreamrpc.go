// Package jstreamrpc implements the mrpc interfaces using the jstream protocol.
// This implementation makes a few design decisions which are not specified in
// the protocol, but which are good best practices none-the-less.
package jstreamrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/mediocregopher/mediocre-go-lib/jstream"
	"github.com/mediocregopher/mediocre-go-lib/mrpc"
)

// TODO Debug
// 	- ReqHead
//		- client encodes into context
//		- handler decodes from context
// 	- ResTail
//		- handler ?
//		- client ?

// TODO Error?
// TODO SizeHints

type debug struct {
	Debug map[string]map[string]json.RawMessage `json:"debug,omitempty"`
}

type reqHead struct {
	debug
	Method string `json:"method"`
}

type resTail struct {
	debug
	Error interface{} `json:"err,omitempty"`
}

type ctxVal int

const (
	ctxValR ctxVal = iota
	ctxValW
)

func unmarshalBody(i interface{}, el jstream.Element) error {
	switch iT := i.(type) {
	case func(*jstream.StreamReader) error:
		stream, err := el.Stream()
		if err != nil {
			return err
		}
		return iT(stream)
	case *io.Reader:
		ioR, err := el.Bytes()
		if err != nil {
			return err
		}
		*iT = ioR
		return nil
	default:
		return el.Value(i)
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
	var head reqHead
	if err := r.Next().Value(&head); err != nil {
		return err
	} else if head.Method == "" {
		return errors.New("request head missing 'method' field")
	}

	var didReadBody bool
	ctx = context.WithValue(ctx, ctxValR, r)
	ctx = context.WithValue(ctx, ctxValW, w)
	ret, err := h.ServeRPC(mrpc.Call{
		Context: ctx,
		Method:  head.Method,
		UnmarshalArgs: func(i interface{}) error {
			didReadBody = true
			return unmarshalBody(i, r.Next())
		},
	})
	// TODO that error is ignored, need a way to differentiate a recoverable
	// error from a non-recoverable one

	// TODO the writing and reading of the next section could be done in
	// parallel?

	// TODO if ret is a byte blob or stream there may be user-spawned
	// go-routines waiting to write to it, but if this errors out before
	// marshalBody is called they will block forever. Probably need to cancel
	// the context to let them know?

	if err := marshalBody(w, ret); err != nil {
		return err
	}

	// TODO to reduce chance of user error maybe Discard should discard any
	// remaining data on the Element, not on Elements which haven't been read
	// from yet. Then it could always be called on the request body at this
	// point.

	// Reading the tail (and maybe discarding the body) should only be done once
	// marshalBody has finished
	if !didReadBody {
		// TODO what if this errors?
		r.Next().Discard()
	}

	// TODO write response tail

	return nil
}
