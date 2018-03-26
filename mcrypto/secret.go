package mcrypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"io"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mlog"
)

// Secret contains a set of bytes which are inteded to remain secret within some
// context (e.g. a backend application keeping a secret from the frontend).
//
// Secret inherently implements the Signer and Verifier interfaces.
//
// Secret can be initialized with NewSecret or NewWeakSecret. The Signatures
// produced by these will be of differing lengths, but either can Verify a
// Signature made by the other as long as the secret bytes they are initialized
// with are the same.
type Secret struct {
	sigSize uint8 // in bytes, shouldn't be more than 32, cause sha256
	secret  []byte

	// only used during tests
	testNow time.Time
}

// NewSecret initializes and returns an instance of Secret which uses the given
// bytes as the underlying secret.
func NewSecret(secret []byte) Secret {
	return Secret{sigSize: 20, secret: secret}
}

// NewWeakSecret is like NewSecret but the Signatures it produces will be
// shorter and weaker (though still secure enough for most applications).
// Signatures produced by either normal or weak Secrets can be Verified by the
// other.
func NewWeakSecret(secret []byte) Secret {
	return Secret{sigSize: 8, secret: secret}
}

func (s Secret) now() time.Time {
	if !s.testNow.IsZero() {
		return s.testNow
	}
	return time.Now()
}

func (s Secret) signRaw(
	r io.Reader,
	sigLen uint8, salt []byte, t time.Time,
) (
	[]byte, error,
) {
	h := hmac.New(sha256.New, s.secret)
	r = sigPrefixReader(r, sigLen, salt, t)
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	return h.Sum(nil)[:sigLen], nil
}

func (s Secret) sign(r io.Reader) (Signature, error) {
	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}

	t := s.now()
	sig, err := s.signRaw(r, s.sigSize, salt, t)
	return Signature{sig: sig, salt: salt, t: t}, err
}

func (s Secret) verify(sig Signature, r io.Reader) error {
	sigB, err := s.signRaw(r, uint8(len(sig.sig)), sig.salt, sig.t)
	if err != nil {
		return mlog.ErrWithKV(err, sig)
	} else if !hmac.Equal(sigB, sig.sig) {
		return mlog.ErrWithKV(ErrInvalidSig, sig)
	}
	return nil
}
