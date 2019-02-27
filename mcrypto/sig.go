package mcrypto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/merr"
)

var (
	errMalformedSig = errors.New("malformed signature")

	// ErrInvalidSig is returned by Signer related functions when an invalid
	// signature is used, e.g. it is a signature for different data, or uses a
	// different secret key, or has expired
	ErrInvalidSig = errors.New("invalid signature")
)

// Signature marshals/unmarshals an actual signature, produced internally by a
// Signer, along with the timestamp the signing took place and a random salt.
//
// All signatures produced in this package will have had the timestamp and salt
// included in the signature's input data, and so are also checked by the
// Verifier.
type Signature struct {
	sig, salt []byte // neither of these should ever be more than 255 bytes long
	t         time.Time
}

// Time returns the timestamp the Signature was generated at
func (s Signature) Time() time.Time {
	return s.t
}

func (s Signature) String() string {
	// ts:8 + saltHeader:1 + salt + sigHeader:1 + sig
	b := make([]byte, 10+len(s.salt)+len(s.sig))
	// It will be year 2286 before the nano doesn't fit in uint64
	binary.BigEndian.PutUint64(b, uint64(s.t.UnixNano()))
	ptr := 8
	b[ptr], ptr = uint8(len(s.salt)), ptr+1
	ptr += copy(b[ptr:], s.salt)
	b[ptr], ptr = uint8(len(s.sig)), ptr+1
	copy(b[ptr:], s.sig)
	return sigV0 + hex.EncodeToString(b)
}

// KV implements the method for the mlog.KVer interface
func (s Signature) KV() map[string]interface{} {
	return map[string]interface{}{"sig": s.String()}
}

// MarshalText implements the method for the encoding.TextMarshaler interface
func (s Signature) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements the method for the encoding.TextUnmarshaler
// interface
func (s *Signature) UnmarshalText(b []byte) error {
	str := string(b)
	strEnc, ok := stripPrefix(str, sigV0)
	if !ok || len(strEnc) < hex.EncodedLen(10) {
		return merr.Wrap(errMalformedSig)
	}

	b, err := hex.DecodeString(strEnc)
	if err != nil {
		return merr.Wrap(err)
	}

	unixNano, b := int64(binary.BigEndian.Uint64(b[:8])), b[8:]
	s.t = time.Unix(0, unixNano).Local()

	readBytes := func() []byte {
		if err != nil {
			return nil
		} else if len(b) < 1+int(b[0]) {
			err = merr.Wrap(errMalformedSig)
			return nil
		}
		out := b[1 : 1+b[0]]
		b = b[1+b[0]:]
		return out
	}

	s.salt = readBytes()
	s.sig = readBytes()
	return err
}

// MarshalJSON implements the method for the json.Marshaler interface
func (s Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements the method for the json.Unmarshaler interface
func (s *Signature) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	return s.UnmarshalText([]byte(str))
}

// returns an io.Reader which will first read out information about the
// Signature which is going to be generated for the data, and then the data from
// the io.Reader itself. When used in conjunction with the Signer/Verifier's
// hashing algorithm this ensures that the other data encoded in the Signature
// (the time and salt) are also encompassed in the sig.
func sigPrefixReader(r io.Reader, sigLen uint8, salt []byte, t time.Time) io.Reader {
	// ts:8 + saltHeader:1 + salt + sigLen:1
	b := make([]byte, 10+len(salt))
	binary.BigEndian.PutUint64(b, uint64(t.UnixNano()))
	b[9] = uint8(len(salt))
	copy(b[9:9+len(salt)], salt)
	b[9+len(salt)] = sigLen
	return io.MultiReader(bytes.NewBuffer(b), r)
}

////////////////////////////////////////////////////////////////////////////////

// Signer is some entity which can generate signatures for arbitrary data and
// can later verify those signatures
type Signer interface {
	sign(io.Reader) (Signature, error)
}

// Verifier is some entity which can verify Signatures produced by a Signer for
// some arbitrary data
type Verifier interface {
	// returns an error if io.Reader returns one ever, or if the Signature
	// couldn't be verified
	verify(Signature, io.Reader) error
}

// Sign reads all data from the io.Reader and signs it using the given Signer
func Sign(s Signer, r io.Reader) (Signature, error) {
	return s.sign(r)
}

// SignBytes uses the Signer to generate a Signature for the given []bytes
func SignBytes(s Signer, b []byte) Signature {
	sig, err := s.sign(bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}
	return sig
}

// SignString uses the Signer to generate a Signature for the given string
func SignString(s Signer, in string) Signature {
	return SignBytes(s, []byte(in))
}

// Verify reads all data from the io.Reader and uses the Verifier to verify that
// the Signature is for that data.
//
// Returns any errors from io.Reader, or ErrInvalidSig (use merr.Equal(err,
// mcrypto.ErrInvalidSig) to check).
func Verify(v Verifier, s Signature, r io.Reader) error {
	return v.verify(s, r)
}

// VerifyBytes uses the Verifier to verify that the Signature is for the given
// []bytes.
//
// Returns ErrInvalidSig (use merr.Equal(err, mcrypto.ErrInvalidSig) to check).
func VerifyBytes(v Verifier, s Signature, b []byte) error {
	return v.verify(s, bytes.NewBuffer(b))
}

// VerifyString uses the Verifier to verify that the Signature is for the given
// string.
//
// Returns ErrInvalidSig (use merr.Equal(err, mcrypto.ErrInvalidSig) to check).
func VerifyString(v Verifier, s Signature, in string) error {
	return VerifyBytes(v, s, []byte(in))
}
