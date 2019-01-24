package merr

import (
	"fmt"
	"path/filepath"
)

// WithValue returns a copy of the original error, automatically wrapping it if
// the error is not from merr (see Wrap). The returned error has a value set on
// with for the given key.
//
// visible determines whether or not the value is visible in the output of
// Error.
func WithValue(e error, k, v interface{}, visible bool) error {
	if e == nil {
		return nil
	}
	er := wrap(e, true, 1)
	er.attr[k] = val{val: v, visible: visible}
	return er
}

// GetValue returns the value embedded in the error for the given key, or nil if
// the error isn't from this package or doesn't have that key embedded.
func GetValue(e error, k interface{}) interface{} {
	if e == nil {
		return nil
	}
	return wrap(e, false, -1).attr[k].val
}

////////////////////////////////////////////////////////////////////////////////

// not really used for attributes, but w/e
const attrKeyErr attrKey = "err"
const attrKeyErrSrc attrKey = "errSrc"

// KVer implements the mlog.KVer interface. This is defined here to avoid this
// package needing to actually import mlog.
type KVer struct {
	kv map[string]interface{}
}

// KV implements the mlog.KVer interface.
func (kv KVer) KV() map[string]interface{} {
	return kv.kv
}

// KV returns a KVer which contains all visible values embedded in the error, as
// well as the original error string itself. Keys will be turned into strings
// using the fmt.Sprint function.
//
// If any keys conflict then their type information will be included as part of
// the key.
func KV(e error) KVer {
	if e == nil {
		return KVer{}
	}

	er := wrap(e, false, 1)
	kvm := make(map[string]interface{}, len(er.attr)+1)

	keys := map[string]interface{}{} // in this case the value is the raw key
	setKey := func(k, v interface{}) {
		kStr := fmt.Sprint(k)
		oldKey := keys[kStr]
		if oldKey == nil {
			keys[kStr] = k
			kvm[kStr] = v
			return
		}

		// check if oldKey is in kvm, if so it needs to be moved to account for
		// its type info
		if oldV, ok := kvm[kStr]; ok {
			delete(kvm, kStr)
			kvm[fmt.Sprintf("%T(%s)", oldKey, kStr)] = oldV
		}

		kvm[fmt.Sprintf("%T(%s)", k, kStr)] = v
	}

	setKey(attrKeyErr, er.err.Error())
	for k, v := range er.attr {
		if !v.visible {
			continue
		}

		stack, ok := v.val.(Stack)
		if !ok {
			setKey(k, v.val)
			continue
		}

		// compress the stack trace to just be the top-most frame
		frame := stack.Frame()
		file, dir := filepath.Base(frame.File), filepath.Dir(frame.File)
		dir = filepath.Base(dir) // only want the first dirname, ie the pkg name
		setKey(attrKeyErrSrc, fmt.Sprintf("%s/%s:%d", dir, file, frame.Line))
	}

	return KVer{kvm}
}
