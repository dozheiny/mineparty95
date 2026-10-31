package netplay

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"mineparty/internal/match"
)

const dialTimeout = 6 * time.Second

// Client is a seat in a room hosted by someone else.
type Client struct {
	conn net.Conn
	w    *wire
	out  chan match.Action
	done chan struct{}
	once sync.Once

	room atomic.Pointer[Room]
	view atomic.Pointer[match.View]
	err  atomic.Pointer[error]
}

// Join connects to the room behind a code.
func Join(code, name string) (*Client, error) {
	addr, token, err := DecodeCode(code)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout("tcp4", addr.String(), dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach the host at %s", addr)
	}
	conn.SetDeadline(time.Now().Add(dialTimeout))
	w := newWire(conn, maxClientFrame)
	if err := w.send(packet{Hello: &hello{Version: ProtocolVersion, Token: token, Name: name}}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("the host dropped the connection: %w", err)
	}
	p, err := w.recv()
	if err != nil {
		conn.Close()
		return nil, errors.New("the host refused the connection; check the room code")
	}
	if p.Reject != "" {
		conn.Close()
		return nil, errors.New(p.Reject)
	}
	if !p.Welcome {
		conn.Close()
		return nil, errors.New("the host sent an unexpected reply")
	}
	conn.SetDeadline(time.Time{})

	c := &Client{conn: conn, w: w, out: make(chan match.Action, 64), done: make(chan struct{})}
	c.room.Store(&Room{You: -1})
	go c.readLoop()
	go c.writeLoop()
	return c, nil
}

func (c *Client) fail(err error) {
	c.err.CompareAndSwap(nil, &err)
	c.Close()
}

func (c *Client) readLoop() {
	for {
		p, err := c.w.recv()
		if err != nil {
			c.fail(errors.New("lost connection to the host"))
			return
		}
		switch {
		case p.Reject != "":
			c.fail(errors.New(p.Reject))
			return
		case p.Room != nil:
			c.room.Store(p.Room)
		case p.View != nil:
			c.view.Store(p.View)
		}
	}
}

func (c *Client) writeLoop() {
	for {
		select {
		case a := <-c.out:
			if err := c.w.send(packet{Action: &a}); err != nil {
				c.fail(errors.New("lost connection to the host"))
				return
			}
		case <-c.done:
			return
		}
	}
}

// Send queues an action for the host, dropping it if the link has stalled.
func (c *Client) Send(a match.Action) {
	select {
	case c.out <- a:
	default:
	}
}

// Room returns the latest lobby state.
func (c *Client) Room() Room { return *c.room.Load() }

// View returns the latest game view, or nil before the first match.
func (c *Client) View() *match.View { return c.view.Load() }

// You returns this player's seat, or -1 before the host has said.
func (c *Client) You() int { return c.room.Load().You }

// Err returns why the session ended, if the host ended it.
func (c *Client) Err() error {
	if e := c.err.Load(); e != nil {
		return *e
	}
	return nil
}

// Close leaves the room.
func (c *Client) Close() {
	c.once.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}
