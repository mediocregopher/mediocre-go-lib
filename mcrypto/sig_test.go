package mcrypto

import (
	. "testing"
	"time"

	"github.com/ansel1/merry"
	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/stretchr/testify/assert"
)

func TestSignerVerifier(t *T) {
	secret := mtest.RandBytes(16)
	sigI, ver := NewSignerVerifier(secret)
	sig := sigI.(signVerifier)
	weakSigI, weakVer := NewWeakSignerVerifier(secret)
	weakSig := weakSigI.(signVerifier)
	var prevStr string
	var prevSig, prevWeakSig Signature
	for i := 0; i < 10000; i++ {
		now := time.Now().Round(0)
		sig.testNow = now
		weakSig.testNow = now

		thisStr := mtest.RandHex(512)
		thisSig := SignString(sig, thisStr)
		thisWeakSig := SignString(weakSig, thisStr)
		thisSigStr, thisWeakSigStr := thisSig.String(), thisWeakSig.String()

		// checking the times made it
		assert.Equal(t, now, thisSig.Time())
		assert.Equal(t, now, thisWeakSig.Time())

		// sanity checks
		assert.NotEmpty(t, thisSigStr)
		assert.NotEmpty(t, thisWeakSigStr)
		assert.NotEqual(t, thisSigStr, thisWeakSigStr)
		assert.True(t, len(thisSigStr) > len(thisWeakSigStr))

		// marshaling/unmarshaling
		var thisSig2, thisWeakSig2 Signature
		assert.NoError(t, thisSig2.UnmarshalText([]byte(thisSigStr)))
		assert.Equal(t, thisSigStr, thisSig2.String())
		assert.NoError(t, thisWeakSig2.UnmarshalText([]byte(thisWeakSigStr)))
		assert.Equal(t, thisWeakSigStr, thisWeakSig2.String())
		assert.Equal(t, now, thisSig2.Time())
		assert.Equal(t, now, thisWeakSig2.Time())

		// Either sigVer should be able to verify either signature
		assert.NoError(t, VerifyString(ver, thisSig, thisStr))
		assert.NoError(t, VerifyString(weakVer, thisWeakSig, thisStr))
		assert.NoError(t, VerifyString(ver, thisWeakSig, thisStr))
		assert.NoError(t, VerifyString(weakVer, thisSig, thisStr))

		if prevStr != "" {
			assert.NotEqual(t, prevSig.String(), thisSigStr)
			assert.NotEqual(t, prevWeakSig.String(), thisWeakSigStr)
			err := VerifyString(ver, prevSig, thisStr)
			assert.True(t, merry.Is(err, ErrInvalidSig))
			err = VerifyString(ver, prevWeakSig, thisStr)
			assert.True(t, merry.Is(err, ErrInvalidSig))
		}
		prevStr = thisStr
		prevSig = thisSig
		prevWeakSig = thisWeakSig
	}
}
