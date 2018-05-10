package mrpc

import "context"

// Debug data is arbitrary data embedded in a Call's request by the Client or in
// its Response by the Server. Debug data is organized into namespaces to help
// avoid conflicts while still preserving serializability.
//
// Debug data is intended to be used for debugging purposes only, and should
// never be used to effect the path-of-action a Call takes. Put another way:
// when implementing a Call always assume that the Debug info has been
// accidentally removed from the Call's request/response.
type Debug map[string]map[string]interface{}

// Copy returns an identical copy of the Debug being called upon
func (d Debug) Copy() Debug {
	d2 := make(Debug, len(d))
	for ns, kv := range d {
		d2[ns] = make(map[string]interface{}, len(kv))
		for k, v := range kv {
			d2[ns][k] = v
		}
	}
	return d2
}

// Set returns a copy of the Debug instance with the key set to the value within
// the given namespace.
func (d Debug) Set(ns, key string, val interface{}) Debug {
	if d == nil {
		return Debug{ns: map[string]interface{}{key, val}}
	}
	d = d.Copy()
	if d[ns] == nil {
		d[ns] = map[string]interface{}{}
	}
	d[ns][key] = val
	return d
}

// Get returns the value for the key within the namespace. Also returns whether
// or not the key was set. This method will never panic.
func (d Debug) Get(ns, key string) (interface{}, bool) {
	if d == nil || d[ns] == nil {
		return nil, false
	}
	val, ok := d[ns][key]
	return val, ok
}

type debugKey int

// CtxWithDebug returns a new Context with the given Debug embedded in it,
// overwriting any previously embedded Debug.
func CtxWithDebug(ctx context.Context, d Debug) context.Context {
	return context.WithValue(ctx, debugKey(0), d)
}

// CtxDebug returns the Debug instance embedded in the Context, or nil if none
// has been embedded.
func CtxDebug(ctx context.Context) Debug {
	d := ctx.Value(debugKey(0))
	if d == nil {
		return Debug(nil)
	}
	return d.(Debug)
}
