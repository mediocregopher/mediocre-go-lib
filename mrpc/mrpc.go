// Package mrpc contains types and functionality to facilitate creating RPC
// interfaces and for making calls against those same interfaces.
//
// This package contains a few fundamental types: Handler, Call, and
// Client. Together these form the components needed to implement nearly any RPC
// system.
//
// TODO document examples
// TODO document Debug?
package mrpc

import (
	"context"
	"fmt"
	"reflect"
)

// Request TODO
type Request struct {
	Context context.Context

	// The name of the RPC method being called.
	Method string

	// Unmarshal takes in a pointer and unmarshals the RPC request's arguments
	// into it. The properties of the unmarshaling are dependent on the
	// underlying implementation of the protocol.
	//
	// This should only be called within ServeRPC.
	Unmarshal func(interface{}) error
}

// ResponseWriter TODO
type ResponseWriter struct {
	Context context.Context

	Respond func(interface{})
	Err     func(error)
}

// Reponse TODO
type Response struct {
	Context   context.Context
	Unmarshal func(interface{}) error
	Err       error
}

// Handler is a type which serves RPC calls. For each incoming Requests the
// ServeRPC method is called with a ResponseWriter which will write the call's
// response back to the client.
type Handler interface {
	ServeRPC(Request, *ResponseWriter)
}

// HandlerFunc can be used to wrap an individual function which fits the
// ServeRPC signature, and use that function as a Handler
type HandlerFunc func(Request, *ResponseWriter)

// ServeRPC implements the method for the Handler interface by calling the
// underlying function
func (hf HandlerFunc) ServeRPC(r Request, rw *ResponseWriter) {
	hf(r, rw)
}

// Client is an entity which can perform RPC calls against a remote endpoint.
//
// res should be a pointer into which the result of the RPC call will be
// unmarshaled according to Client's implementation. args will be marshaled and
// sent to the remote endpoint according to Client's implementation.
type Client interface {
	CallRPC(ctx context.Context, method string, args interface{}) Response
}

// ClientFunc can be used to wrap an individual function which fits the CallRPC
// signature, and use that function as a Client
type ClientFunc func(context.Context, string, interface{}) Response

// CallRPC implements the method for the Client interface by calling the
// underlying function
func (cf ClientFunc) CallRPC(
	ctx context.Context,
	method string,
	args interface{},
) Response {
	return cf(ctx, method, args)
}

// ReflectClient returns a Client whose CallRPC method will use reflection to
// call the given Handler's ServeRPC method directly, using reflect.Value's Set
// method to copy CallRPC's args parameter into the Request's Unmarshal method's
// receiver parameter, and similarly to copy the result from ServeRPC into
// the Response's Unmarshal method's receiver parameter.
func ReflectClient(h Handler) Client {
	into := func(dst, src interface{}) error {
		dstV, srcV := reflect.ValueOf(dst), reflect.ValueOf(src)
		dstVi, srcVi := reflect.Indirect(dstV), reflect.Indirect(srcV)
		if !dstVi.CanSet() || dstVi.Type() != srcVi.Type() {
			return fmt.Errorf("can't set value of type %v into type %v", srcV.Type(), dstV.Type())
		}
		dstVi.Set(srcVi)
		return nil
	}

	return ClientFunc(func(
		ctx context.Context,
		method string,
		args interface{},
	) Response {
		req := Request{
			Context:   ctx,
			Method:    method,
			Unmarshal: func(i interface{}) error { return into(i, args) },
		}
		var res interface{}
		var resErr error
		rw := ResponseWriter{
			Context: context.Background(),
			Respond: func(i interface{}) { res = i },
			Err:     func(err error) { resErr = err },
		}

		h.ServeRPC(req, &rw)

		return Response{
			Context: rw.Context,
			Unmarshal: func(i interface{}) error {
				if resErr != nil {
					return resErr
				}
				return into(i, res)
			},
			Err: resErr,
		}
	})
}
