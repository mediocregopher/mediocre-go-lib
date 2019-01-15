package mcrypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mlog"
)

var (
	errMalformedPublicKey  = errors.New("malformed public key")
	errMalformedPrivateKey = errors.New("malformed private key")
)

// NewKeyPair generates and returns a complementary public/private key pair
func NewKeyPair() (PublicKey, PrivateKey) {
	return newKeyPair(2048)
}

// NewWeakKeyPair is like NewKeyPair but the returned pair uses fewer bits
// (though still a reasonably secure amount for data that doesn't need security
// guarantees into the year 3000 whatever).
func NewWeakKeyPair() (PublicKey, PrivateKey) {
	return newKeyPair(1024)
}

func newKeyPair(bits int) (PublicKey, PrivateKey) {
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		panic(err)
	}
	return PublicKey{priv.PublicKey}, PrivateKey{priv}
}

////////////////////////////////////////////////////////////////////////////////

// PublicKey is a wrapper around an rsa.PublicKey which simplifies using it and
// adds marshaling/unmarshaling methods.
//
// A PublicKey automatically implements the Verifier interface.
type PublicKey struct {
	rsa.PublicKey
}

func (pk PublicKey) verify(s Signature, r io.Reader) error {
	h := sha256.New()
	r = sigPrefixReader(r, 32, s.salt, s.t)
	if _, err := io.Copy(h, r); err != nil {
		return err
	}
	if err := rsa.VerifyPSS(&pk.PublicKey, crypto.SHA256, h.Sum(nil), s.sig, nil); err != nil {
		return mlog.ErrWithKV(ErrInvalidSig, s)
	}
	return nil
}

func (pk PublicKey) String() string {
	nB := pk.N.Bytes()
	b := make([]byte, 8+len(nB))
	// the exponent is never negative so this is fine
	binary.BigEndian.PutUint64(b, uint64(pk.E))
	copy(b[8:], nB)
	return pubKeyV0 + hex.EncodeToString(b)
}

// KV implements the method for the mlog.KVer interface
func (pk PublicKey) KV() map[string]interface{} {
	return map[string]interface{}{"publicKey": pk.String()}
}

// MarshalText implements the method for the encoding.TextMarshaler interface
func (pk PublicKey) MarshalText() ([]byte, error) {
	return []byte(pk.String()), nil
}

// UnmarshalText implements the method for the encoding.TextUnmarshaler
// interface
func (pk *PublicKey) UnmarshalText(b []byte) error {
	str := string(b)
	strEnc, ok := stripPrefix(str, pubKeyV0)
	if !ok || len(strEnc) <= hex.EncodedLen(8) {
		return mlog.ErrWithKV(errMalformedPublicKey, mlog.KV{"pubKeyStr": str})
	}

	b, err := hex.DecodeString(strEnc)
	if err != nil {
		return mlog.ErrWithKV(err, mlog.KV{"pubKeyStr": str})
	}

	pk.E = int(binary.BigEndian.Uint64(b))
	pk.N = new(big.Int)
	pk.N.SetBytes(b[8:])
	return nil
}

// MarshalJSON implements the method for the json.Marshaler interface
func (pk PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(pk.String())
}

// UnmarshalJSON implements the method for the json.Unmarshaler interface
func (pk *PublicKey) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return pk.UnmarshalText([]byte(s))
}

////////////////////////////////////////////////////////////////////////////////

// PrivateKey is a wrapper around an rsa.PrivateKey which simplifies using it
// and adds marshaling/unmarshaling methods.
//
// A PrivateKey automatically implements the Signer interface.
type PrivateKey struct {
	*rsa.PrivateKey
}

func (pk PrivateKey) sign(r io.Reader) (Signature, error) {
	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}
	t := time.Now()
	h := sha256.New()
	// sigLen has to be 32 here (bytes returned by sha256) cause of the way the
	// VerifyPSS function is
	if _, err := io.Copy(h, sigPrefixReader(r, 32, salt, t)); err != nil {
		return Signature{}, err
	}
	sig, err := rsa.SignPSS(rand.Reader, pk.PrivateKey, crypto.SHA256, h.Sum(nil), nil)
	return Signature{sig: sig, salt: salt, t: t}, err
}

func (pk PrivateKey) String() string {
	numBytes := binary.MaxVarintLen64 * 3 // public exponent, N, and D
	nB, dB := pk.PublicKey.N.Bytes(), pk.D.Bytes()
	numBytes += len(nB) + len(dB)

	primes := make([][]byte, len(pk.Primes))
	for i, prime := range pk.Primes {
		primes[i] = prime.Bytes()
		numBytes += binary.MaxVarintLen64 + len(primes[i])
	}

	b, ptr := make([]byte, numBytes), 0
	ptr += binary.PutUvarint(b[ptr:], uint64(pk.E))
	ptr += binary.PutUvarint(b[ptr:], uint64(len(nB)))
	ptr += copy(b[ptr:], nB)
	ptr += binary.PutUvarint(b[ptr:], uint64(len(dB)))
	ptr += copy(b[ptr:], dB)

	for _, prime := range primes {
		ptr += binary.PutUvarint(b[ptr:], uint64(len(prime)))
		ptr += copy(b[ptr:], prime)
	}

	return privKeyV0 + hex.EncodeToString(b[:ptr])
}

// KV implements the method for the mlog.KVer interface
func (pk PrivateKey) KV() map[string]interface{} {
	return map[string]interface{}{"privateKey": pk.String()}
}

// MarshalText implements the method for the encoding.TextMarshaler interface
func (pk PrivateKey) MarshalText() ([]byte, error) {
	return []byte(pk.String()), nil
}

// UnmarshalText implements the method for the encoding.TextUnmarshaler
// interface
func (pk *PrivateKey) UnmarshalText(b []byte) error {
	str := string(b)
	strEnc, ok := stripPrefix(str, privKeyV0)
	if !ok {
		return mlog.ErrWithKV(errMalformedPrivateKey, mlog.KV{"privKeyStr": str})
	}

	b, err := hex.DecodeString(strEnc)
	if err != nil {
		return mlog.ErrWithKV(err, mlog.KV{"privKeyStr": str})
	}

	e, n := binary.Uvarint(b)
	if n <= 0 {
		return mlog.ErrWithKV(errMalformedPrivateKey, mlog.KV{"privKeyStr": str})
	}
	pk.PublicKey.E = int(e)
	b = b[n:]

	bigInt := func() *big.Int {
		if err != nil {
			return nil
		}
		l, n := binary.Uvarint(b)
		if n <= 0 {
			err = errMalformedPrivateKey
		}
		b = b[n:]
		i := new(big.Int)
		i.SetBytes(b[:l])
		b = b[l:]
		return i
	}

	pk.PublicKey.N = bigInt()
	pk.D = bigInt()
	for len(b) > 0 && err == nil {
		pk.Primes = append(pk.Primes, bigInt())
	}

	if err != nil {
		return mlog.ErrWithKV(err, mlog.KV{"privKeyStr": str})
	}
	return nil
}

// MarshalJSON implements the method for the json.Marshaler interface
func (pk PrivateKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(pk.String())
}

// UnmarshalJSON implements the method for the json.Unmarshaler interface
func (pk *PrivateKey) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return pk.UnmarshalText([]byte(s))
}
