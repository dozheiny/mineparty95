package netplay

import (
	"compress/flate"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"mineparty/internal/match"
)

// ProtocolVersion must match on both ends of a connection.
const ProtocolVersion = 3

// Frame limits. Clients only send small messages, so a host refuses anything
// bigger than this before allocating memory for it.
const (
	maxHostFrame   = 4 << 10
	maxClientFrame = 1 << 20
	writeTimeout   = 5 * time.Second
)

// packet is the one message type on the wire; exactly one field is set.
type packet struct {
	Hello   *hello        `json:",omitempty"`
	Welcome bool          `json:",omitempty"`
	Reject  string        `json:",omitempty"`
	Action  *match.Action `json:",omitempty"`
	Room    *Room         `json:",omitempty"`
	View    *match.View   `json:",omitempty"`
}

type hello struct {
	Version int
	Token   [4]byte
	Name    string
}

// wire frames JSON messages over one long compressed stream. Consecutive
// game views are nearly identical, so the compressor's shared history makes
// each one cost only a few bytes more than what actually changed.
type wire struct {
	conn  net.Conn
	limit int
	fr    io.ReadCloser
	mu    sync.Mutex
	fw    *flate.Writer
}

func newWire(conn net.Conn, limit int) *wire {
	fw, err := flate.NewWriter(conn, flate.BestSpeed)
	if err != nil {
		panic(err) // BestSpeed is always a valid level
	}
	return &wire{conn: conn, limit: limit, fr: flate.NewReader(conn), fw: fw}
}

func (w *wire) send(p packet) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(data)))
	if _, err := w.fw.Write(size[:]); err != nil {
		return err
	}
	if _, err := w.fw.Write(data); err != nil {
		return err
	}
	return w.fw.Flush()
}

func (w *wire) recv() (packet, error) {
	var p packet
	var size [4]byte
	if _, err := io.ReadFull(w.fr, size[:]); err != nil {
		return p, err
	}
	n := binary.BigEndian.Uint32(size[:])
	if n > uint32(w.limit) {
		return p, fmt.Errorf("frame of %d bytes exceeds the %d byte limit", n, w.limit)
	}
	data := make([]byte, n)
	if _, err := io.ReadFull(w.fr, data); err != nil {
		return p, err
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return p, errors.New("malformed message")
	}
	return p, nil
}
