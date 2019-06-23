package mcfg

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/stretchr/testify/assert"
)

func TestPopulate(t *T) {
	{
		cmp := new(mcmp.Component)
		a := Int(cmp, "a")
		cmpFoo := cmp.Child("foo")
		b := Int(cmpFoo, "b")
		c := Int(cmpFoo, "c")
		d := Int(cmp, "d", ParamDefault(4))

		err := Populate(cmp, &SourceCLI{
			Args: []string{"--a=1", "--foo-b=2"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, *a)
		assert.Equal(t, 2, *b)
		assert.Equal(t, 0, *c)
		assert.Equal(t, 4, *d)
	}

	{ // test that required params are enforced
		cmp := new(mcmp.Component)
		a := Int(cmp, "a")
		cmpFoo := cmp.Child("foo")
		b := Int(cmpFoo, "b")
		c := Int(cmpFoo, "c", ParamRequired())

		err := Populate(cmp, &SourceCLI{
			Args: []string{"--a=1", "--foo-b=2"},
		})
		assert.Error(t, err)

		err = Populate(cmp, &SourceCLI{
			Args: []string{"--a=1", "--foo-b=2", "--foo-c=3"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, *a)
		assert.Equal(t, 2, *b)
		assert.Equal(t, 3, *c)
	}
}

func TestParamDefaultOrRequired(t *T) {
	{
		cmp := new(mcmp.Component)
		Int(cmp, "a", ParamDefaultOrRequired(0))
		params := CollectParams(cmp)
		assert.Equal(t, "a", params[0].Name)
		assert.Equal(t, true, params[0].Required)
		assert.Equal(t, new(int), params[0].Into)
	}
	{
		cmp := new(mcmp.Component)
		Int(cmp, "a", ParamDefaultOrRequired(1))
		i := 1
		params := CollectParams(cmp)
		assert.Equal(t, "a", params[0].Name)
		assert.Equal(t, false, params[0].Required)
		assert.Equal(t, &i, params[0].Into)
	}
}
