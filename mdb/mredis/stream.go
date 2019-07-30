package mredis

import (
	"bufio"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"

	"github.com/mediocregopher/radix/v3"
	"github.com/mediocregopher/radix/v3/resp/resp2"
)

// borrowed from radix
type streamReaderEntry struct {
	stream  []byte
	entries []radix.StreamEntry
}

func (s *streamReaderEntry) UnmarshalRESP(br *bufio.Reader) error {
	var ah resp2.ArrayHeader
	if err := ah.UnmarshalRESP(br); err != nil {
		return err
	}
	if ah.N != 2 {
		return errors.New("invalid xread[group] response")
	}

	var stream resp2.BulkStringBytes
	stream.B = s.stream[:0]
	if err := stream.UnmarshalRESP(br); err != nil {
		return err
	}
	s.stream = stream.B

	return (resp2.Any{I: &s.entries}).UnmarshalRESP(br)
}

// StreamEntry wraps radix's StreamEntry type in order to provde some extra
// functionality.
type StreamEntry struct {
	radix.StreamEntry

	// Ack is used in order to acknowledge that a stream message has been
	// successfully consumed and should not be consumed again.
	Ack func() error

	// Nack is used to declare that a stream message was not successfully
	// consumed and it needs to be consumed again.
	Nack func()
}

// StreamOpts are options used to initialize a Stream instance. Fields are
// required unless otherwise noted.
type StreamOpts struct {
	// Key is the redis key at which the redis stream resides.
	Key string

	// Group is the name of the consumer group which will consume from Key.
	Group string

	// Consumer is the name of this particular consumer. This value should
	// remain the same across restarts of the process.
	Consumer string

	// (Optional) InitialCursor is only used when the consumer group is first
	// being created, and indicates where in the stream the consumer group
	// should start consuming from.
	//
	// "0" indicates the group should consume from the start of the stream. "$"
	// indicates the group should not consume any old messages, only those added
	// after the group is initialized. A specific message id can be given to
	// consume only those messages with greater ids.
	//
	// Defaults to "$".
	InitialCursor string

	// (Optional) ReadCount indicates the max number of messages which should be
	// read on every XREADGROUP call. 0 indicates no limit.
	ReadCount int

	// (Optional) Block indicates what BLOCK value is sent to XREADGROUP calls.
	// This value _must_ be less than the ReadtTimeout the redis client is
	// using.
	//
	// Defaults to 5 * time.Second
	Block time.Duration
}

func (opts *StreamOpts) fillDefaults() {
	if opts.InitialCursor == "" {
		opts.InitialCursor = "$"
	}
	if opts.Block == 0 {
		opts.Block = 5 * time.Second
	}
}

// Stream wraps a Redis instance in order to provide an abstraction over
// consuming messages from a single redis stream. Stream is intended to be used
// in a single-threaded manner, and doesn't spawn any go-routines.
//
// See https://redis.io/topics/streams-intro
type Stream struct {
	client *Redis
	opts   StreamOpts

	// entries are stored to buf in id decreasing order, and then read from it
	// from back-to-front. This allows us to not have to re-allocate the buffer
	// during runtime.
	buf []StreamEntry

	hasInit    bool
	numPending int64
}

// NewStream initializes and returns a Stream instance using the given options.
func NewStream(r *Redis, opts StreamOpts) *Stream {
	opts.fillDefaults()
	return &Stream{
		client: r,
		opts:   opts,
		buf:    make([]StreamEntry, 0, opts.ReadCount),
	}
}

func (s *Stream) init() error {
	// MKSTREAM is not documented, but will make the stream if it doesn't
	// already exist. Only the most elite redis gurus know of it's
	// existence, don't tell anyone.
	err := s.client.Do(radix.Cmd(nil, "XGROUP", "CREATE", s.opts.Key, s.opts.Group, s.opts.InitialCursor, "MKSTREAM"))
	if err == nil {
		// cool
	} else if errStr := err.Error(); !strings.HasPrefix(errStr, `BUSYGROUP Consumer Group name already exists`) {
		return merr.Wrap(err, s.client.cmp.Context())
	}

	// if we're here it means init succeeded, mark as such and gtfo
	s.hasInit = true
	return nil
}

func (s *Stream) wrapEntry(entry radix.StreamEntry) StreamEntry {
	return StreamEntry{
		StreamEntry: entry,
		Ack: func() error {
			return s.client.Do(radix.Cmd(nil, "XACK", s.opts.Key, s.opts.Group, entry.ID.String()))
		},
		Nack: func() { atomic.AddInt64(&s.numPending, 1) },
	}
}

func (s *Stream) fillBufFrom(id string) error {
	args := []string{"GROUP", s.opts.Group, s.opts.Consumer}
	if s.opts.ReadCount > 0 {
		args = append(args, "COUNT", strconv.Itoa(s.opts.ReadCount))
	}
	args = append(args, "BLOCK", strconv.Itoa(int(s.opts.Block.Seconds())))
	args = append(args, "STREAMS", s.opts.Key, id)

	var srEntries []streamReaderEntry
	err := s.client.Do(radix.Cmd(&srEntries, "XREADGROUP", args...))
	if err != nil {
		return merr.Wrap(err, s.client.cmp.Context())
	} else if len(srEntries) == 0 {
		return nil // no messages
	} else if len(srEntries) != 1 || string(srEntries[0].stream) != s.opts.Key {
		return merr.New("malformed return from XREADGROUP",
			mctx.Annotate(s.client.cmp.Context(), "srEntries", srEntries))
	}
	entries := srEntries[0].entries

	for i := len(entries) - 1; i >= 0; i-- {
		s.buf = append(s.buf, s.wrapEntry(entries[i]))
	}
	return nil
}

func (s *Stream) fillBuf() error {
	if len(s.buf) > 0 {
		return nil
	} else if !s.hasInit {
		if err := s.init(); err != nil {
			return err
		} else if !s.hasInit {
			return nil
		}
	}

	numPending := atomic.LoadInt64(&s.numPending)
	if numPending > 0 {
		if err := s.fillBufFrom("0"); err != nil {
			return err
		} else if len(s.buf) > 0 {
			return nil
		}

		// no pending entries, we can mark Stream as such and continue. This
		// _might_ fail if another routine called Nack in between originally
		// loading numPending and now, in which case we should leave the buffer
		// alone and let it get filled again later.
		if !atomic.CompareAndSwapInt64(&s.numPending, numPending, 0) {
			return nil
		}
	}

	return s.fillBufFrom(">")
}

// Next returns the next StreamEntry which needs processing, or false. This
// method is expected to block for up to the value of the Block field in
// StreamOpts.
//
// If an error is returned it's up to the caller whether or not they want to
// keep retrying.
func (s *Stream) Next() (StreamEntry, bool, error) {
	if err := s.fillBuf(); err != nil {
		return StreamEntry{}, false, err
	} else if len(s.buf) == 0 {
		return StreamEntry{}, false, nil
	}

	l := len(s.buf)
	entry := s.buf[l-1]
	s.buf = s.buf[:l-1]
	return entry, true, nil
}
