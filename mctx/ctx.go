// Package mctx extends the builtin context package to organize Contexts into a
// hierarchy.
//
// All functions and methods in this package are thread-safe unless otherwise
// noted.
//
// Parents and children
//
// Each node in the hierarchy is given a name and is aware of all of its
// ancestors. The sequence of ancestor's names, ending in the node's name, is
// called its "path". For example:
//
//	ctx := context.Background()
//	ctxA := mctx.NewChild(ctx, "A")
//	ctxB := mctx.NewChild(ctx, "B")
//	fmt.Printf("ctx:%#v\n", mctx.Path(ctx)) // prints "ctx:[]string(nil)"
//	fmt.Printf("ctxA:%#v\n", mctx.Path(ctxA)) // prints "ctxA:[]string{"A"}
//	fmt.Printf("ctxB:%#v\n", mctx.Path(ctxB)) // prints "ctxA:[]string{"A", "B"}
//
// WithChild can be used to incorporate a child into its parent, making the
// parent's children iterable on it:
//
//	ctx := context.Background()
//	ctxA1 := mctx.NewChild(ctx, "A1")
//	ctxA2 := mctx.NewChild(ctx, "A2")
//	ctx = mctx.WithChild(ctx, ctxA1)
//	ctx = mctx.WithChild(ctx, ctxA2)
//	for _, childCtx := range mctx.Children(ctx) {
//		fmt.Printf("%q\n", mctx.Name(childCtx)) // prints "A1" then "A2"
//	}
//
// Key/Value
//
// The context's key/value namespace is split into two: a space local to a node,
// not inherited from its parent or inheritable by its children
// (WithLocalValue), and the original one which comes with the builtin context
// package (context.WithValue):
//
//	ctx := context.Background()
//	ctx = context.WithValue(ctx, "inheritableKey", "foo")
//	ctx = mctx.WithLocalValue(ctx, "localKey", "bar")
//	childCtx := mctx.NewChild(ctx, "child")
//
//	// ctx.Value("inheritableKey") == "foo"
//	// child.Value("inheritableKey") == "foo"
//	// mctx.LocalValue(ctx, "localKey") == "bar"
//	// mctx.LocalValue(child, "localKey") == nil
//
// Annotations
//
// Annotations are a special case of local key/values, where the data being
// stored is specifically runtime metadata which would be useful for logging,
// error output, etc... Annotation data might include an IP address of a
// connected client, a userID the client has authenticated as, the primary key
// of a row in a database being queried, etc...
//
// Annotations are always tied to the path of the node they were set on, so that
// even when the annotations of two contexts are merged the annotation data will
// not overlap unless the contexts have the same path.
package mctx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

////////////////////////////////////////////////////////////////////////////////

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

// Children returns all children of this Context which have been added by
// WithChild, in the order they were added. If this Context wasn't produced by
// WithChild then this returns nil.
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

func pathHash(path []string) string {
	pathHash := sha256.New()
	for _, pathEl := range path {
		fmt.Fprintf(pathHash, "%q.", pathEl)
	}
	return hex.EncodeToString(pathHash.Sum(nil))
}

// Name returns the name this Context was created with via NewChild, or false if
// this Context was not created via NewChild.
func Name(ctx context.Context) (string, bool) {
	path := Path(ctx)
	if len(path) == 0 {
		return "", false
	}
	return path[len(path)-1], true
}

// NewChild creates and returns a new Context based off of the parent one.  The
// child will have a path which is the parent's path appended with the given
// name. In order for the parent to "see" the child (via the Child or Children
// functions) the WithChild function must be used.
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
// Children list. If the child's name is already taken in the parent then this
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
// children, etc... in a breadth-first order. If the callback returns false then
// the function returns without visiting any more Contexts.
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

// WithLocalValue is like context.WithValue, but the stored value will not be
// present on any children created via NewChild. Local values must be retrieved
// with the LocalValue function in this package. Local values share a different
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
