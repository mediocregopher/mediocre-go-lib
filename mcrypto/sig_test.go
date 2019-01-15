package mcrypto

import (
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/stretchr/testify/assert"
)

func TestSignerVerifier(t *T) {
	secret := NewSecret(mrand.Bytes(16))
	var prevStr string
	var prevSig Signature
	for i := 0; i < 10000; i++ {
		now := time.Now().Round(0)
		secret.testNow = now

		thisStr := mrand.Hex(512)
		thisSig := SignString(secret, thisStr)
		thisSigStr := thisSig.String()

		// sanity checks
		assert.NotEmpty(t, thisSigStr)
		assert.Equal(t, now, thisSig.Time())
		assert.NoError(t, VerifyString(secret, thisSig, thisStr))

		// marshaling/unmarshaling
		var thisSig2 Signature
		assert.NoError(t, thisSig2.UnmarshalText([]byte(thisSigStr)))
		assert.Equal(t, thisSigStr, thisSig2.String())
		assert.Equal(t, now, thisSig2.Time())
		assert.NoError(t, VerifyString(secret, thisSig2, thisStr))

		if prevStr != "" {
			assert.NotEqual(t, prevSig.String(), thisSigStr)
			err := VerifyString(secret, prevSig, thisStr)
			assert.True(t, merr.Equal(err, ErrInvalidSig))
		}
		prevStr = thisStr
		prevSig = thisSig
	}
}
