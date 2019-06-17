package mcmp

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestComponent(t *T) {
	assertValue := func(c *Component, key, expectedValue interface{}) massert.Assertion {
		val := c.Value(key)
		ok := c.HasValue(key)
		return massert.All(
			massert.Equal(expectedValue, val),
			massert.Equal(expectedValue != nil, ok),
		)
	}

	assertName := func(c *Component, expectedName string) massert.Assertion {
		name, ok := c.Name()
		return massert.All(
			massert.Equal(expectedName, name),
			massert.Equal(expectedName != "", ok),
		)
	}

	// test that a Component is initialized correctly
	c := new(Component)
	massert.Require(t,
		assertName(c, ""),
		massert.Length(c.Path(), 0),
		massert.Length(c.Children(), 0),
		assertValue(c, "foo", nil),
		assertValue(c, "bar", nil),
	)

	// test that setting values work, and that values aren't inherited
	c.SetValue("foo", 1)
	child := c.Child("child")
	massert.Require(t,
		assertName(child, "child"),
		massert.Equal([]string{"child"}, child.Path()),
		massert.Length(child.Children(), 0),
		massert.Equal([]*Component{child}, c.Children()),
		assertValue(c, "foo", 1),
		assertValue(child, "foo", nil),
	)

	// test that a child setting a value does not affect the parent
	child.SetValue("bar", 2)
	massert.Require(t,
		assertValue(c, "bar", nil),
		assertValue(child, "bar", 2),
	)

	assertInheritedValue := func(c *Component, key, expectedValue interface{}) massert.Assertion {
		val, ok := c.InheritedValue(key)
		return massert.All(
			massert.Equal(expectedValue, val),
			massert.Equal(expectedValue != nil, ok),
		)
	}

	// test that InheritedValue does what it's supposed to
	massert.Require(t,
		assertInheritedValue(c, "foo", 1),
		assertInheritedValue(child, "foo", 1),
		assertInheritedValue(c, "bar", nil),
		assertInheritedValue(child, "bar", 2),
		assertInheritedValue(c, "xxx", nil),
		assertInheritedValue(child, "xxx", nil),
	)
}
func TestBreadFirstVisit(t *T) {
	cmp := new(Component)
	cmp1 := cmp.Child("1")
	cmp1a := cmp1.Child("a")
	cmp1b := cmp1.Child("b")
	cmp2 := cmp.Child("2")

	{
		got := make([]*Component, 0, 5)
		BreadthFirstVisit(cmp, func(cmp *Component) bool {
			got = append(got, cmp)
			return true
		})
		massert.Require(t,
			massert.Equal([]*Component{cmp, cmp1, cmp2, cmp1a, cmp1b}, got),
		)
	}

	{
		got := make([]*Component, 0, 3)
		BreadthFirstVisit(cmp, func(cmp *Component) bool {
			if len(cmp.Path()) > 1 {
				return false
			}
			got = append(got, cmp)
			return true
		})
		massert.Require(t,
			massert.Equal([]*Component{cmp, cmp1, cmp2}, got),
		)
	}
}
