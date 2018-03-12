package mcrypto

import (
	"strings"
	. "testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUID(t *T) {
	var prevT time.Time
	var prev UUID
	for i := 0; i < 10000; i++ {
		thisT := time.Now().Round(0)        // strip monotonic clock
		require.True(t, thisT.After(prevT)) // sanity check
		this := NewUUID(thisT)

		// basic
		assert.True(t, strings.HasPrefix(this.String(), uuidV0))

		// comparisons with prev
		assert.False(t, prev.Equal(this))
		assert.NotEqual(t, prev.String(), this.String())
		assert.True(t, this.String() > prev.String())
		prev = this

		// check time unpacking
		assert.Equal(t, thisT, this.Time())

		// check marshal/unmarshal
		thisStr, err := this.MarshalText()
		require.NoError(t, err)
		var this2 UUID
		require.NoError(t, this2.UnmarshalText(thisStr))
		assert.True(t, this.Equal(this2), "this:%q this2:%q", this, this2)
	}
}
