// Package mrpc contains types and functionality to facilitate creating RPC
// interfaces and for making calls against those same interfaces
//
// This package contains a few fundamental types: Handler, Call, and
// Client. Together these form the components needed to implement nearly any RPC
// system.
//
// TODO an example of an implementation of these interfaces can be found in the
// m package
package mrpc

import (
	"context"
	"fmt"
	"reflect"
)

// Handler is a type which serves RPC calls. For each incoming Call the ServeRPC
// method is called, and the return from the method is used as the response. If
// an error is returned the response return is ignored.
type Handler interface {
	ServeRPC(Call) (interface{}, error)
}

// HandlerFunc can be used to wrap an individual function which fits the
// ServeRPC signature, and use that function as a Handler
type HandlerFunc func(Call) (interface{}, error)

// ServeRPC implements the method for the Handler interface by calling the
// underlying function
func (hf HandlerFunc) ServeRPC(c Call) (interface{}, error) {
	return hf(c)
}

// Call is passed into the ServeRPC method and contains all information about
// the incoming RPC call which is being made
type Call interface {
	Context() context.Context

	// Method returns the name of the RPC method being called
	Method() string

	// UnmarshalArgs takes in a pointer and unmarshals the RPC call's arguments
	// into it. The properties of the unmarshaling are dependent on the
	// underlying implementation of the codec types.
	UnmarshalArgs(interface{}) error
}

type call struct {
	ctx           context.Context
	method        string
	unmarshalArgs func(interface{}) error
}

func (c call) Context() context.Context {
	return c.ctx
}

func (c call) Method() string {
	return c.method
}

func (c call) UnmarshalArgs(i interface{}) error {
	return c.unmarshalArgs(i)
}

// WithContext returns the same Call it's given, but the new Call will return
// the given context when Context() is called
func WithContext(c Call, ctx context.Context) Call {
	return call{ctx: ctx, method: c.Method(), unmarshalArgs: c.UnmarshalArgs}
}

// WithMethod returns the same Call it's given, but the new Call will return the
// given method name when Method() is called
func WithMethod(c Call, method string) Call {
	return call{ctx: c.Context(), method: method, unmarshalArgs: c.UnmarshalArgs}
}

// Client is an entity which can perform RPC calls against a remote endpoint.
//
// res should be a pointer into which the result of the RPC call will be
// unmarshaled according to Client's implementation. args will be marshaled and
// sent to the remote endpoint according to Client's implementation.
type Client interface {
	CallRPC(ctx context.Context, res interface{}, method string, args interface{}) error
}

// ClientFunc can be used to wrap an individual function which fits the CallRPC
// signature, and use that function as a Client
type ClientFunc func(context.Context, interface{}, string, interface{}) error

// CallRPC implements the method for the Client interface by calling the
// underlying function
func (cf ClientFunc) CallRPC(
	ctx context.Context,
	res interface{},
	method string,
	args interface{},
) error {
	return cf(ctx, res, method, args)
}

// ReflectClient returns a Client whose CallRPC method will use reflection to
// call the given Handler's ServeRPC method directly, using reflect.Value's Set
// method to copy CallRPC's args parameter into UnmarshalArgs' receiver
// parameter, and similarly to copy the result from ServeRPC into CallRPC's
// receiver parameter.
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
		resInto interface{},
		method string,
		args interface{},
	) error {
		c := call{
			ctx:           ctx,
			method:        method,
			unmarshalArgs: func(i interface{}) error { return into(i, args) },
		}

		res, err := h.ServeRPC(c)
		if err != nil {
			return err
		}

		return into(resInto, res)
	})
}
