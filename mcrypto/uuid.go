package mcrypto

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mlog"
)

var errMalformedUUID = errors.New("malformed UUID string")

// UUID is a universally unique identifier which embeds within it a timestamp.
//
// Only Unmarshal methods should be called on the zero UUID value.
//
// Comparing the equality of two UUID's should always be done using the Equal
// method, or by comparing their string forms.
//
// The string form of UUIDs (returned by String or MarshalText) are
// lexigraphically order-able by their embedded timestamp.
type UUID struct {
	b []byte
}

// NewUUID populates and returns a new UUID instance which embeds the given time
func NewUUID(t time.Time) UUID {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[:8], uint64(t.UnixNano()))
	if _, err := rand.Read(b[8:]); err != nil {
		panic(err)
	}
	return UUID{b: b}
}

func (u UUID) String() string {
	return uuidV0 + hex.EncodeToString(u.b)
}

// Equal returns whether or not the two UUID's are the same value
func (u UUID) Equal(u2 UUID) bool {
	return bytes.Equal(u.b, u2.b)
}

// Time unpacks and returns the timestamp embedded in the UUID
func (u UUID) Time() time.Time {
	unixNano := int64(binary.BigEndian.Uint64(u.b[:8]))
	return time.Unix(0, unixNano).Local()
}

// KV implements the method for the mlog.KVer interface
func (u UUID) KV() map[string]interface{} {
	return map[string]interface{}{"uuid": u.String()}
}

// MarshalText implements the method for the encoding.TextMarshaler interface
func (u UUID) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// UnmarshalText implements the method for the encoding.TextUnmarshaler
// interface
func (u *UUID) UnmarshalText(b []byte) error {
	str := string(b)
	strEnc, ok := stripPrefix(str, uuidV0)
	if !ok || len(strEnc) != hex.EncodedLen(16) {
		return mlog.ErrWithKV(errMalformedUUID, mlog.KV{"uuidStr": str})
	}
	b, err := hex.DecodeString(strEnc)
	if err != nil {
		return mlog.ErrWithKV(err, mlog.KV{"uuidStr": str})
	}
	u.b = b
	return nil
}

// MarshalJSON implements the method for the json.Marshaler interface
func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// UnmarshalJSON implements the method for the json.Unmarshaler interface
func (u *UUID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return u.UnmarshalText([]byte(s))
}
