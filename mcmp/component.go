package mcmp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
)

type child struct {
	*Component
	name string
}

// Component describes a single component of a program, and holds onto
// key/values for that component for use in generic libraries which instantiate
// those components.
//
// When instantiating a component it's generally necessary to know where in the
// component hierarchy it lies, for purposes of creating configuration
// parameters and so-forth. To support this, Components are able to spawn of
// child Components, each with a blank key/value namespace. Each child is
// differentiated from the other by a name, and a Component is able to use its
// Path (the sequence of names of its ancestors) to differentiate itself from
// any other component in the hierarchy.
//
// A new Component, i.e. the root Component in the hierarchy, can be initialized
// by doing:
//	new(Component).
//
// Method's on Component are thread-safe.
type Component struct {
	l sync.RWMutex

	path     []string
	parent   *Component
	children []child

	kv  map[interface{}]interface{}
	ctx context.Context
}

// SetValue sets the given key to the given value on the Component, overwriting
// any previous value for that key.
func (c *Component) SetValue(key, value interface{}) {
	c.l.Lock()
	defer c.l.Unlock()
	if c.kv == nil {
		c.kv = make(map[interface{}]interface{}, 1)
	}
	c.kv[key] = value
}

func (c *Component) value(key interface{}) (interface{}, bool) {
	c.l.RLock()
	defer c.l.RUnlock()
	if c.kv == nil {
		return nil, false
	}
	value, ok := c.kv[key]
	return value, ok
}

// Value returns the value which has been set for the given key.
func (c *Component) Value(key interface{}) interface{} {
	value, _ := c.value(key)
	return value
}

// InheritedValue returns the value which has been set for the given key. It first
// looks for the key on the receiver Component. If not found, it will look on
// its parent Component, and so on, until the key is found. If the key is not
// found on the root Component then false is returned.
func (c *Component) InheritedValue(key interface{}) (interface{}, bool) {
	value, ok := c.value(key)
	if ok {
		return value, ok
	} else if c.parent == nil {
		return nil, false
	}
	return c.parent.InheritedValue(key)
}

// Values returns all key/value pairs which have been set via SetValue.
func (c *Component) Values() map[interface{}]interface{} {
	c.l.RLock()
	defer c.l.RUnlock()
	out := make(map[interface{}]interface{}, len(c.kv))
	for k, v := range c.kv {
		out[k] = v
	}
	return out
}

// HasValue returns true if the given key has had a value set on it with
// SetValue.
func (c *Component) HasValue(key interface{}) bool {
	c.l.RLock()
	defer c.l.RUnlock()
	_, ok := c.kv[key]
	return ok
}

// Child returns a new child component of the method receiver. The child will
// have the given name, and its Path will be the receiver's path with the name
// appended. The child will not inherit any of the receiver's key/value pairs.
//
// If a child of the given name has already been created this method will panic.
func (c *Component) Child(name string) *Component {
	c.l.Lock()
	defer c.l.Unlock()
	for _, child := range c.children {
		if child.name == name {
			panic(fmt.Sprintf("child with name %q already exists", name))
		}
	}

	childComp := &Component{
		path:   append(c.path, name),
		parent: c,
	}
	c.children = append(c.children, child{name: name, Component: childComp})
	return childComp
}

// Children returns all Components created via the Child method on this
// Component, in the order they were created.
func (c *Component) Children() []*Component {
	c.l.RLock()
	defer c.l.RUnlock()
	children := make([]*Component, len(c.children))
	for i := range c.children {
		children[i] = c.children[i].Component
	}
	return children
}

// Name returns the name this Component was created with (via the Child method),
// or false if this Component was not created via Child (and is therefore the
// root Component).
func (c *Component) Name() (string, bool) {
	c.l.RLock()
	defer c.l.RUnlock()
	if len(c.path) == 0 {
		return "", false
	}
	return c.path[len(c.path)-1], true
}

// Path returns the sequence of names which were passed into Child calls in
// order to create this Component. If the Component was not created via Child
// (and is therefore the root Component) this will return an empty slice.
//
//	root := new(Component)
//	child := root.Child("child")
//	grandChild := child.Child("grandchild")
//	fmt.Printf("%#v\n", root.Path())     	// "[]string(nil)"
//	fmt.Printf("%#v\n", child.Path())       // []string{"child"}
//	fmt.Printf("%#v\n", grandChild.Path())  // []string{"child", "grandchild"}
//
func (c *Component) Path() []string {
	c.l.RLock()
	defer c.l.RUnlock()
	return c.path
}

func (c *Component) pathStr() string {
	path := make([]string, len(c.path))
	copy(path, c.path)
	for i := range path {
		path[i] = strings.ReplaceAll(path[i], "/", `\/`)
	}
	return "/" + strings.Join(path, "/")
}

type annotateKey string

func (c *Component) getCtx() context.Context {
	if c.ctx == nil {
		c.ctx = mctx.Annotated(annotateKey("componentPath"), c.pathStr())
	}
	return c.ctx
}

// Annotate annotates the Component's internal Context in-place, such that they
// will be included in any future calls to the Context method.
func (c *Component) Annotate(kv ...interface{}) {
	c.l.Lock()
	defer c.l.Unlock()
	c.ctx = mctx.Annotate(c.getCtx(), kv...)
}

// Context returns a Context which has been annotated with any annotations from
// Annotate calls to this Component, as well as some default annotations which
// are always included.
func (c *Component) Context() context.Context {
	c.l.Lock()
	defer c.l.Unlock()
	return c.getCtx()
}

// BreadthFirstVisit visits this Component and all of its children, and their
// children, etc... in a breadth-first order. If the callback returns false then
// the function returns without visiting any more Components.
func BreadthFirstVisit(c *Component, callback func(*Component) bool) {
	queue := []*Component{c}
	for len(queue) > 0 {
		if !callback(queue[0]) {
			return
		}
		for _, child := range queue[0].Children() {
			queue = append(queue, child)
		}
		queue = queue[1:]
	}
}
