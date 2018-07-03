package mcrypto

import (
	. "testing"
	"time"

	"github.com/ansel1/merry"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/stretchr/testify/assert"
)

func TestSecretSignVerify(t *T) {
	secretRaw := mrand.Bytes(16)
	secret := NewSecret(secretRaw)
	weakSecret := NewWeakSecret(secretRaw)
	var prevStr string
	var prevSig, prevWeakSig Signature
	for i := 0; i < 10000; i++ {
		now := time.Now().Round(0)
		secret.testNow = now
		weakSecret.testNow = now

		thisStr := mrand.Hex(512)
		thisSig := SignString(secret, thisStr)
		thisWeakSig := SignString(weakSecret, thisStr)
		thisSigStr, thisWeakSigStr := thisSig.String(), thisWeakSig.String()

		// sanity checks
		assert.Equal(t, now, thisSig.Time())
		assert.Equal(t, now, thisWeakSig.Time())
		assert.NotEmpty(t, thisSigStr)
		assert.NotEmpty(t, thisWeakSigStr)
		assert.NotEqual(t, thisSigStr, thisWeakSigStr)
		assert.True(t, len(thisSigStr) > len(thisWeakSigStr))

		// Either secret should be able to verify either signature
		assert.NoError(t, VerifyString(secret, thisSig, thisStr))
		assert.NoError(t, VerifyString(weakSecret, thisWeakSig, thisStr))
		assert.NoError(t, VerifyString(secret, thisWeakSig, thisStr))
		assert.NoError(t, VerifyString(weakSecret, thisSig, thisStr))

		if prevStr != "" {
			assert.NotEqual(t, prevSig.String(), thisSigStr)
			assert.NotEqual(t, prevWeakSig.String(), thisWeakSigStr)
			err := VerifyString(secret, prevSig, thisStr)
			assert.True(t, merry.Is(err, ErrInvalidSig))
			err = VerifyString(secret, prevWeakSig, thisStr)
			assert.True(t, merry.Is(err, ErrInvalidSig))
		}
		prevStr = thisStr
		prevSig = thisSig
		prevWeakSig = thisWeakSig
	}
}
