package mcrypto

import (
	. "testing"
	"time"

	"github.com/ansel1/merry"
	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/stretchr/testify/assert"
)

func TestSigner(t *T) {
	secret := mtest.RandBytes(16)
	signer, weakSigner := NewSigner(secret), NewWeakSigner(secret)
	var prevStr, prevSig, prevWeakSig string
	for i := 0; i < 10000; i++ {
		thisStr := mtest.RandHex(512)
		thisSig := SignString(signer, thisStr)
		thisWeakSig := SignString(weakSigner, thisStr)

		// sanity checks
		assert.NotEqual(t, thisSig, thisWeakSig)
		assert.True(t, len(thisSig) > len(thisWeakSig))

		// Either signer should be able to verify either signature
		assert.NoError(t, VerifyString(signer, thisSig, thisStr))
		assert.NoError(t, VerifyString(weakSigner, thisWeakSig, thisStr))
		assert.NoError(t, VerifyString(signer, thisWeakSig, thisStr))
		assert.NoError(t, VerifyString(weakSigner, thisSig, thisStr))

		if prevStr != "" {
			assert.NotEqual(t, prevSig, thisSig)
			assert.NotEqual(t, prevWeakSig, thisWeakSig)
			err := VerifyString(signer, prevSig, thisStr)
			assert.True(t, merry.Is(err, ErrInvalidSig))
			err = VerifyString(signer, prevWeakSig, thisStr)
			assert.True(t, merry.Is(err, ErrInvalidSig))
		}
		prevStr = thisStr
		prevSig = thisSig
		prevWeakSig = thisWeakSig
	}
}

func TestExpireSigner(t *T) {
	origNow := time.Now()
	s := ExpireSigner(NewSigner(mtest.RandBytes(16)), 1*time.Hour).(expireSigner)
	s.testNow = origNow
	str := mtest.RandHex(32)
	sig := SignString(s, str)

	// in the immediate the sig should obviously work
	assert.NoError(t, VerifyString(s, sig, str))
	err := VerifyString(s, sig, mtest.RandHex(32))
	assert.True(t, merry.Is(err, ErrInvalidSig))

	// within the timeout it should still work
	s.testNow = s.testNow.Add(1 * time.Minute)
	assert.NoError(t, VerifyString(s, sig, str))

	// but a new "now" should then generate a different sig
	sig2 := SignString(s, str)
	assert.NotEqual(t, sig, sig2)
	assert.NoError(t, VerifyString(s, sig2, str))

	// jumping forward an hour should expire the first sig, but not the second
	s.testNow = s.testNow.Add(1 * time.Hour)
	err = VerifyString(s, sig, str)
	assert.True(t, merry.Is(err, ErrInvalidSig))
	assert.NoError(t, VerifyString(s, sig2, str))
}

func TestUniqueSigner(t *T) {
	s := UniqueSigner(NewSigner(mtest.RandBytes(16)))
	str := mtest.RandHex(32)
	sigA, sigB := SignString(s, str), SignString(s, str)
	assert.NotEqual(t, sigA, sigB)
	assert.NoError(t, VerifyString(s, sigA, str))
	assert.NoError(t, VerifyString(s, sigB, str))
}
