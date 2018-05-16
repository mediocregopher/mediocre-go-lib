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

// Request describes an RPC request being processed by a Handler
type Request struct {
	Context context.Context

	// The name of the RPC method being called.
	Method string

	// Unmarshal takes in a pointer and unmarshals the RPC request's arguments
	// into it. The properties of the unmarshaling are dependent on the
	// underlying implementation of the protocol.
	Unmarshal func(interface{}) error

	// Debugging information being carried with the Request. See Debug's docs
	// for more on how it is intended to be used.
	Debug Debug
}

// ResponseWriter is used to capture the response of an RPC request being
// processed by a Handler
type ResponseWriter struct {
	// Response should be overwritten with whatever response to the call should
	// be. The exact nature and behavior of how the response value is treated is
	// dependent on the RPC implementation.
	Response interface{}

	// Debug may be overwritten to provide debugging information back to the
	// Client with the Response. See Debug's docs for more on how it is intended
	// to be used.
	Debug Debug
}

// Reponse describes the response from the RPC call being returned to the
// Client.
type Response struct {
	// Unmarshal takes in a pointer value into which the Client will unmarshal
	// the response value. The exact nature and behavior of how the pointer
	// value is treated is dependend on the RPC implementation.
	Unmarshal func(interface{}) error

	// Debug will be whatever debug information was set by the server when
	// responding to the call.
	Debug Debug
}

// Handler is a type which serves RPC calls. For each incoming Requests the
// ServeRPC method is called with a ResponseWriter which will write the call's
// response back to the client.
//
// Any go-routines spawned by ServeRPC should expect to terminate if the
// Request's Context is canceled
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
type Client interface {
	CallRPC(ctx context.Context, method string, args interface{}, debug Debug) Response
}

// ClientFunc can be used to wrap an individual function which fits the CallRPC
// signature, and use that function as a Client
type ClientFunc func(context.Context, string, interface{}, Debug) Response

// CallRPC implements the method for the Client interface by calling the
// underlying function
func (cf ClientFunc) CallRPC(
	ctx context.Context,
	method string,
	args interface{},
	debug Debug,
) Response {
	return cf(ctx, method, args, debug)
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
		debug Debug,
	) Response {
		req := Request{
			Context:   ctx,
			Method:    method,
			Unmarshal: func(i interface{}) error { return into(i, args) },
			Debug:     debug,
		}
		rw := ResponseWriter{}
		h.ServeRPC(req, &rw)

		return Response{
			Unmarshal: func(i interface{}) error {
				return into(i, rw.Response)
			},
			Debug: rw.Debug,
		}
	})
}
