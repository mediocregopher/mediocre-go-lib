package mtime

import (
	. "testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuration(t *T) {
	{
		b, err := Duration{5 * time.Second}.MarshalText()
		assert.NoError(t, err)
		assert.Equal(t, []byte("5s"), b)

		var d Duration
		assert.NoError(t, d.UnmarshalText(b))
		assert.Equal(t, 5*time.Second, d.Duration)
	}

	{
		b, err := Duration{5 * time.Second}.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, []byte(`"5s"`), b)

		var d Duration
		assert.NoError(t, d.UnmarshalJSON(b))
		assert.Equal(t, 5*time.Second, d.Duration)
	}
}
