// Package mctx extends the builtin context package to organize Contexts into a
// hierarchy. Each node in the hierarchy is given a name and is aware of all of
// its ancestors.
//
// This package also provides extra functionality which allows contexts
// to be more useful when used in the hierarchy.
//
// All functions and methods in this package are thread-safe unless otherwise
// noted.
package mctx

import (
	"context"
	"fmt"
)

////////////////////////////////////////////////////////////////////////////////

// New returns a new context which can be used as the root context for all
// purposes in this framework.
//func New() Context {
//	return &context{Context: goctx.Background()}
//}

type ancestryKey int // 0 -> children, 1 -> parent, 2 -> path

const (
	ancestryKeyChildren ancestryKey = iota
	ancestryKeyChildrenMap
	ancestryKeyParent
	ancestryKeyPath
)

// Child returns the Context of the given name which was added to parent via
// WithChild, or nil if no Context of that name was ever added.
func Child(parent context.Context, name string) context.Context {
	childrenMap, _ := parent.Value(ancestryKeyChildrenMap).(map[string]int)
	if len(childrenMap) == 0 {
		return nil
	}
	i, ok := childrenMap[name]
	if !ok {
		return nil
	}
	return parent.Value(ancestryKeyChildren).([]context.Context)[i]
}

// Children returns all children of this Context which have been kept by
// WithChild, mapped by their name. If this Context wasn't produced by WithChild
// then this returns nil.
func Children(parent context.Context) []context.Context {
	children, _ := parent.Value(ancestryKeyChildren).([]context.Context)
	return children
}

func childrenCP(parent context.Context) ([]context.Context, map[string]int) {
	children := Children(parent)
	// plus 1 because this is most commonly used in WithChild, which will append
	// to it. At any rate it doesn't hurt anything.
	outChildren := make([]context.Context, len(children), len(children)+1)
	copy(outChildren, children)

	childrenMap, _ := parent.Value(ancestryKeyChildrenMap).(map[string]int)
	outChildrenMap := make(map[string]int, len(childrenMap)+1)
	for name, i := range childrenMap {
		outChildrenMap[name] = i
	}

	return outChildren, outChildrenMap
}

// parentOf returns the Context from which this one was generated via NewChild.
// Returns nil if this Context was not generated via NewChild.
//
// This is kept private because the behavior is a bit confusing. This will
// return the Context which was passed into NewChild, but users would probably
// expect it to return the one from WithChild if they were to call this.
func parentOf(ctx context.Context) context.Context {
	parent, _ := ctx.Value(ancestryKeyParent).(context.Context)
	return parent
}

// Path returns the sequence of names which were used to produce this Context
// via the NewChild function. If this Context wasn't produced by NewChild then
// this returns nil.
func Path(ctx context.Context) []string {
	path, _ := ctx.Value(ancestryKeyPath).([]string)
	return path
}

func pathCP(ctx context.Context) []string {
	path := Path(ctx)
	// plus 1 because this is most commonly used in NewChild, which will append
	// to it. At any rate it doesn't hurt anything.
	outPath := make([]string, len(path), len(path)+1)
	copy(outPath, path)
	return outPath
}

// Name returns the name this Context was generated with via NewChild, or false
// if this Context was not generated via NewChild.
func Name(ctx context.Context) (string, bool) {
	path := Path(ctx)
	if len(path) == 0 {
		return "", false
	}
	return path[len(path)-1], true
}

// NewChild creates a new Context based off of the parent one, and returns a new
// instance of the passed in parent and the new child. The child will have a
// path which is the parent's path with the given name appended. The parent will
// have the new child as part of its set of children (see Children function).
//
// If the parent already has a child of the given name this function panics.
func NewChild(parent context.Context, name string) context.Context {
	if Child(parent, name) != nil {
		panic(fmt.Sprintf("child with name %q already exists on parent", name))
	}

	childPath := append(pathCP(parent), name)
	child := withoutLocalValues(parent)
	child = context.WithValue(child, ancestryKeyChildren, nil)    // unset children
	child = context.WithValue(child, ancestryKeyChildrenMap, nil) // unset children
	child = context.WithValue(child, ancestryKeyParent, parent)
	child = context.WithValue(child, ancestryKeyPath, childPath)
	return child
}

func isChild(parent, child context.Context) bool {
	parentPath, childPath := Path(parent), Path(child)
	if len(parentPath) != len(childPath)-1 {
		return false
	}

	for i := range parentPath {
		if parentPath[i] != childPath[i] {
			return false
		}
	}
	return true
}

// WithChild returns a modified parent which holds a reference to child in its
// Children set. If the child's name is already taken in the parent then this
// function panics.
func WithChild(parent, child context.Context) context.Context {
	if !isChild(parent, child) {
		panic(fmt.Sprintf("child cannot be kept by Context which is not its parent"))
	}

	name, _ := Name(child)
	children, childrenMap := childrenCP(parent)
	if _, ok := childrenMap[name]; ok {
		panic(fmt.Sprintf("child with name %q already exists on parent", name))
	}
	children = append(children, child)
	childrenMap[name] = len(children) - 1

	parent = context.WithValue(parent, ancestryKeyChildren, children)
	parent = context.WithValue(parent, ancestryKeyChildrenMap, childrenMap)
	return parent
}

// BreadthFirstVisit visits this Context and all of its children, and their
// children, in a breadth-first order. If the callback returns false then the
// function returns without visiting any more Contexts.
func BreadthFirstVisit(ctx context.Context, callback func(context.Context) bool) {
	queue := []context.Context{ctx}
	for len(queue) > 0 {
		if !callback(queue[0]) {
			return
		}
		for _, child := range Children(queue[0]) {
			queue = append(queue, child)
		}
		queue = queue[1:]
	}
}

////////////////////////////////////////////////////////////////////////////////
// local value stuff

type localValsKey int

type localVal struct {
	prev     *localVal
	key, val interface{}
}

// WithLocalValue is like WithValue, but the stored value will not be present
// on any children created via WithChild. Local values must be retrieved with
// the LocalValue function in this package. Local values share a different
// namespace than the normal WithValue/Value values (i.e. they do not overlap).
func WithLocalValue(ctx context.Context, key, val interface{}) context.Context {
	prev, _ := ctx.Value(localValsKey(0)).(*localVal)
	return context.WithValue(ctx, localValsKey(0), &localVal{
		prev: prev,
		key:  key, val: val,
	})
}

func withoutLocalValues(ctx context.Context) context.Context {
	return context.WithValue(ctx, localValsKey(0), nil)
}

// LocalValue returns the value for the given key which was set by a call to
// WithLocalValue, or nil if no value was set for the given key.
func LocalValue(ctx context.Context, key interface{}) interface{} {
	lv, _ := ctx.Value(localValsKey(0)).(*localVal)
	for {
		if lv == nil {
			return nil
		} else if lv.key == key {
			return lv.val
		}
		lv = lv.prev
	}
}

func localValuesIter(ctx context.Context, callback func(key, val interface{})) {
	m := map[interface{}]struct{}{}
	lv, _ := ctx.Value(localValsKey(0)).(*localVal)
	for {
		if lv == nil {
			return
		} else if _, ok := m[lv.key]; !ok {
			callback(lv.key, lv.val)
			m[lv.key] = struct{}{}
		}
		lv = lv.prev
	}
}

// LocalValues returns all key/value pairs which have been set on the Context
// via WithLocalValue.
func LocalValues(ctx context.Context) map[interface{}]interface{} {
	m := map[interface{}]interface{}{}
	localValuesIter(ctx, func(key, val interface{}) {
		m[key] = val
	})
	return m
}
