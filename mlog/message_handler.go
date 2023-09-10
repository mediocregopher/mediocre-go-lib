package mlog

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sync"

	"github.com/mediocregopher/mediocre-go-lib/v2/mctx"
)

// MessageHandler is a type which can process Messages in some way.
//
// NOTE that Logger does not handle thread-safety, that must be done inside the
// MessageHandler if necessary.
type MessageHandler interface {
	Handle(FullMessage) error

	// Sync flushes any buffered data to the handler's output, e.g. a file or
	// network connection. If the handler doesn't buffer data then this will be
	// a no-op.
	Sync() error
}

func maybeSyncWriter(w io.Writer) error {
	if s, ok := w.(interface{ Sync() error }); ok {
		return s.Sync()
	} else if f, ok := w.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type msgHandler struct {
	l   sync.Mutex
	out io.Writer
	aa  mctx.Annotations
}

// NewMessageHandler initializes and returns a MessageHandler which will write
// all messages to the given io.Writer in a human-readable format.
//
// If the io.Writer also implements a Sync or Flush method then that will be
// called when Sync is called on the returned MessageHandler.
func NewMessageHandler(out io.Writer) MessageHandler {
	return &msgHandler{
		out: out,
		aa:  mctx.Annotations{},
	}
}

func (h *msgHandler) Sync() error {
	h.l.Lock()
	defer h.l.Unlock()
	return maybeSyncWriter(h.out)
}

func (h *msgHandler) Handle(msg FullMessage) error {
	h.l.Lock()
	defer h.l.Unlock()

	var namespaceStr string

	if len(msg.Namespace) > 0 {
		namespaceStr = "[" + path.Join(msg.Namespace...) + "] "
	}

	var annotationsStr string

	if ss := mctx.EvaluateAnnotations(msg.Context, h.aa).StringSlice(true); len(ss) > 0 {
		for i := range ss {
			annotationsStr += fmt.Sprintf(" %q=%q", ss[i][0], ss[i][1])
		}
	}

	fmt.Fprintf(
		h.out, "%s %s%s%s\n",
		msg.Level.String(),
		namespaceStr,
		msg.Description,
		annotationsStr,
	)

	for k := range h.aa {
		delete(h.aa, k)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type jsonMsgHandler struct {
	l   sync.Mutex
	out io.Writer
	enc *json.Encoder
	aa  mctx.Annotations
}

// NewJSONMessageHandler initializes and returns a MessageHandler which will
// write all messages to the given io.Writer as JSON objects.
//
// If the io.Writer also implements a Sync or Flush method then that will be
// called when Sync is called on the returned MessageHandler.
func NewJSONMessageHandler(out io.Writer) MessageHandler {
	return &jsonMsgHandler{
		out: out,
		enc: json.NewEncoder(out),
		aa:  mctx.Annotations{},
	}
}

type messageJSON struct {
	TimeDate    string   `json:"td"`
	Timestamp   int64    `json:"ts"`
	Level       string   `json:"level"`
	Namespace   []string `json:"ns,omitempty"`
	Description string   `json:"descr"`
	LevelInt    int      `json:"level_int"`

	// key -> value
	Annotations map[string]string `json:"annotations,omitempty"`
}

const msgTimeFormat = "06/01/02 15:04:05.000000"

func (h *jsonMsgHandler) Handle(msg FullMessage) error {
	h.l.Lock()
	defer h.l.Unlock()

	msgJSON := messageJSON{
		TimeDate:    msg.Time.UTC().Format(msgTimeFormat),
		Timestamp:   msg.Time.UnixNano(),
		Level:       msg.Level.String(),
		LevelInt:    msg.Level.Int(),
		Namespace:   msg.Namespace,
		Description: msg.Description,
		Annotations: mctx.EvaluateAnnotations(msg.Context, h.aa).StringMap(),
	}

	for k := range h.aa {
		delete(h.aa, k)
	}

	return h.enc.Encode(msgJSON)
}

func (h *jsonMsgHandler) Sync() error {
	h.l.Lock()
	defer h.l.Unlock()
	return maybeSyncWriter(h.out)
}
