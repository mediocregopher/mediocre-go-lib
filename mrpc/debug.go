package mrpc

// Debug data is arbitrary data embedded in a Request by the Client or in its
// Response by the Server. Debug data is organized into namespaces to help avoid
// conflicts while still preserving serializability.
//
// Debug data is intended to be used for debugging purposes only, and should
// never be used to effect the path-of-action a call takes. Put another way:
// when implementing a call always assume that the Debug info has been
// accidentally removed from the call's Request/Response.
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
// the given namespace. If Debug is nil a new instance is created and returned
// with the key set.
func (d Debug) Set(ns, key string, val interface{}) Debug {
	if d == nil {
		return Debug{ns: map[string]interface{}{key: val}}
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
