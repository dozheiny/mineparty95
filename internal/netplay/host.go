package netplay

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"mineparty/internal/engine"
	"mineparty/internal/match"
)

// Tuning for the host.
const (
	MaxPlayers       = 4
	DefaultPort      = 37195
	DefaultBotDelay  = 500 * time.Millisecond
	handshakeTimeout = 5 * time.Second
	tickEvery        = 30 * time.Millisecond
	heartbeat        = 100 * time.Millisecond
	maxActionsPerSec = 120
	maxNameRunes     = 14
)

var botNames = []string{"Ada", "Turing", "Grace", "Hopper", "Lovelace", "Knuth"}

// RoomPlayer is one entry in the lobby.
type RoomPlayer struct {
	Name string
	Bot  bool
	Wins int
	Left bool // disconnected; the seat clears at the next start
}

// Room is the lobby state everyone can see.
type Room struct {
	You     int
	Players []RoomPlayer
	Mode    match.Mode
	Board   engine.Config
	InMatch bool
}

// Session is a seat in a game, whether hosting, joined or offline.
type Session interface {
	Send(match.Action)
	Room() Room
	View() *match.View
	You() int
	Err() error
	Close()
}

type seat struct {
	name   string
	wins   int
	bot    *match.Bot
	nextAt time.Duration
	peer   *peer
	gone   bool
}

// HostOptions configures NewHost.
type HostOptions struct {
	Name     string
	Port     int  // TCP port to try first; 0 picks any free port
	Loopback bool // listen on 127.0.0.1 only
	Offline  bool // no listening at all, for solo and CPU games
}

// Host owns the match. Everything that touches game state runs on one
// goroutine, reached through the mail channel, so the rules need no locks.
type Host struct {
	opts     HostOptions
	token    [4]byte
	listener net.Listener
	port     int
	epoch    time.Time

	mail chan func()
	done chan struct{}
	once sync.Once

	// Owned by the run goroutine.
	seats      []*seat
	cfg        match.Config
	botDelay   time.Duration
	game       *match.Match
	recorded   bool
	round      int
	dirty      bool
	lastPublic time.Duration
	botCount   int

	room atomic.Pointer[Room]
	view atomic.Pointer[match.View]

	discPort atomic.Int32
	disc     atomic.Pointer[net.UDPConn]

	internet internetState
}

// NewHost creates a room with the local player in seat 0.
func NewHost(opts HostOptions) (*Host, error) {
	h := &Host{
		opts:     opts,
		epoch:    time.Now(),
		mail:     make(chan func(), 256),
		done:     make(chan struct{}),
		botDelay: DefaultBotDelay,
		cfg:      match.Config{Mode: match.Race, Board: engine.Intermediate},
	}
	if _, err := rand.Read(h.token[:]); err != nil {
		return nil, err
	}
	h.seats = []*seat{{name: cleanName(opts.Name)}}
	if !opts.Offline {
		if err := h.listen(); err != nil {
			return nil, err
		}
		go h.acceptLoop()
	}
	h.publishRoom()
	go h.run()
	return h, nil
}

func (h *Host) listen() error {
	bind := ""
	if h.opts.Loopback {
		bind = "127.0.0.1"
	}
	l, err := net.Listen("tcp4", net.JoinHostPort(bind, fmt.Sprint(h.opts.Port)))
	if err != nil {
		// The preferred port is busy, perhaps with a second copy of the game.
		l, err = net.Listen("tcp4", net.JoinHostPort(bind, "0"))
	}
	if err != nil {
		return fmt.Errorf("can't open a port for hosting: %w", err)
	}
	h.listener = l
	h.port = l.Addr().(*net.TCPAddr).Port
	return nil
}

func cleanName(name string) string {
	var out []rune
	for _, r := range strings.TrimSpace(name) {
		if len(out) == maxNameRunes {
			break
		}
		if unicode.IsPrint(r) {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "Player"
	}
	return string(out)
}

func (h *Host) now() time.Duration { return time.Since(h.epoch) }

// do queues f on the game goroutine.
func (h *Host) do(f func()) {
	select {
	case h.mail <- f:
	case <-h.done:
	}
}

// call runs f on the game goroutine and waits for it.
func (h *Host) call(f func()) {
	finished := make(chan struct{})
	h.do(func() {
		f()
		close(finished)
	})
	select {
	case <-finished:
	case <-h.done:
	}
}

func (h *Host) run() {
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()
	for {
		select {
		case f := <-h.mail:
			f()
		case <-ticker.C:
			h.tick()
		case <-h.done:
			return
		}
	}
}

func (h *Host) tick() {
	if h.game == nil {
		return
	}
	now := h.now()
	h.game.Tick(now)
	for i, s := range h.seats {
		if s.bot == nil || s.gone || now < s.nextAt {
			continue
		}
		if a, ok := s.bot.Next(h.game.View(i, now)); ok {
			// Moving the cursor first shows teammates what the CPU is doing.
			h.game.Apply(now, match.Action{Player: i, Kind: match.Cursor, Tile: a.Tile})
			h.game.Apply(now, a)
			h.dirty = true
		}
		s.nextAt = now + h.botDelay
	}
	if h.dirty || now-h.lastPublic >= heartbeat {
		h.publishViews(now)
	}
}

func (h *Host) publishViews(now time.Duration) {
	h.dirty = false
	h.lastPublic = now
	h.recordResult()
	for i, s := range h.seats {
		if s.gone || s.bot != nil {
			continue
		}
		v := h.game.View(i, now)
		if s.peer != nil {
			s.peer.queueView(&v)
		} else {
			h.view.Store(&v)
		}
	}
}

func (h *Host) recordResult() {
	if h.recorded || h.game.Phase() != match.Over {
		return
	}
	h.recorded = true
	v := h.game.View(0, h.now())
	if v.Winner >= 0 && v.Winner < len(h.seats) {
		h.seats[v.Winner].wins++
	}
	h.publishRoom()
}

func (h *Host) inMatch() bool { return h.game != nil && h.game.Phase() != match.Over }

func (h *Host) roomFor(you int) *Room {
	r := &Room{You: you, Mode: h.cfg.Mode, Board: h.cfg.Board, InMatch: h.inMatch()}
	for _, s := range h.seats {
		r.Players = append(r.Players, RoomPlayer{Name: s.name, Bot: s.bot != nil, Wins: s.wins, Left: s.gone})
	}
	return r
}

func (h *Host) publishRoom() {
	h.room.Store(h.roomFor(0))
	for i, s := range h.seats {
		if s.peer != nil && !s.gone {
			s.peer.queueRoom(h.roomFor(i))
		}
	}
}

// Send performs an action for the local player.
func (h *Host) Send(a match.Action) {
	h.do(func() {
		if h.game == nil {
			return
		}
		a.Player = 0
		now := h.now()
		h.game.Apply(now, a)
		h.dirty = true
		// The local player gets their result right away; remote players
		// are coalesced into the next tick.
		h.recordResult()
		v := h.game.View(0, now)
		h.view.Store(&v)
	})
}

// Room returns the latest lobby state.
func (h *Host) Room() Room { return *h.room.Load() }

// View returns the latest game view, or nil before the first match.
func (h *Host) View() *match.View { return h.view.Load() }

// You returns the local player's seat, which is always the first.
func (h *Host) You() int { return 0 }

// Offline reports whether this room accepts no connections.
func (h *Host) Offline() bool { return h.opts.Offline }

// Err always returns nil: a host can't lose its connection to itself.
func (h *Host) Err() error { return nil }

// Configure sets the mode and board for the next match.
func (h *Host) Configure(mode match.Mode, board engine.Config) {
	h.call(func() {
		h.cfg.Mode, h.cfg.Board = mode, board
		h.publishRoom()
	})
}

// SetBotDelay sets how long CPU players pause between moves.
func (h *Host) SetBotDelay(d time.Duration) {
	h.call(func() { h.botDelay = d })
}

// AddBot adds a CPU player if there is a free seat.
func (h *Host) AddBot() {
	h.call(func() {
		if h.occupied() >= MaxPlayers || h.inMatch() {
			return
		}
		name := "CPU " + botNames[h.botCount%len(botNames)]
		seed := uint64(h.botCount) + uint64(time.Now().UnixNano())
		h.botCount++
		h.seats = append(h.seats, &seat{name: name, bot: match.NewBot(len(h.seats), seed)})
		h.publishRoom()
	})
}

// Kick removes a seat other than the host's own while in the lobby.
func (h *Host) Kick(index int) {
	h.call(func() {
		if index <= 0 || index >= len(h.seats) || h.inMatch() {
			return
		}
		if p := h.seats[index].peer; p != nil {
			p.reject("The host removed you from the room.")
		}
		h.vacate(index)
		h.publishRoom()
	})
}

// vacate frees a seat between matches. Seat numbers shift, so the finished
// match is dropped too, as its players no longer line up with the seats.
func (h *Host) vacate(index int) {
	h.removeSeat(index)
	h.game = nil
}

func (h *Host) removeSeat(index int) {
	h.seats = append(h.seats[:index], h.seats[index+1:]...)
	for i, s := range h.seats[index:] {
		if s.bot != nil {
			s.bot.SetPlayer(index + i)
		}
	}
}

// Start begins a match, or a rematch, with everyone in the room.
func (h *Host) Start() error {
	return h.begin(nil)
}

// StartWith begins a match on an explicit configuration. Solo games use this
// for the seeds and fixed openings of daily challenges.
func (h *Host) StartWith(cfg match.Config) error {
	return h.begin(&cfg)
}

func (h *Host) begin(override *match.Config) error {
	var result error
	h.call(func() {
		for i := len(h.seats) - 1; i >= 0; i-- {
			if h.seats[i].gone {
				h.removeSeat(i)
			}
		}
		cfg := h.cfg
		if override != nil {
			cfg = *override
		}
		lo, hi := cfg.Mode.PlayerLimits()
		if n := len(h.seats); n < lo || n > hi {
			result = fmt.Errorf("%v needs %s, and the room has %d", cfg.Mode, playerRange(lo, hi), n)
			return
		}
		cfg.Players = nil
		for _, s := range h.seats {
			cfg.Players = append(cfg.Players, s.name)
		}
		if cfg.Seed == 0 {
			var seed [8]byte
			rand.Read(seed[:])
			cfg.Seed = binary.LittleEndian.Uint64(seed[:])
		}
		h.round++
		cfg.Round = h.round
		now := h.now()
		h.game = match.New(cfg, now)
		h.recorded = false
		for _, s := range h.seats {
			s.nextAt = 0
		}
		h.publishRoom()
		h.publishViews(now)
	})
	return result
}

func playerRange(lo, hi int) string {
	if lo == hi {
		return fmt.Sprintf("exactly %d players", lo)
	}
	return fmt.Sprintf("%d to %d players", lo, hi)
}

// Close shuts the room down and disconnects everyone.
func (h *Host) Close() {
	h.once.Do(func() {
		h.closeInternet()
		h.call(func() {
			for _, s := range h.seats {
				if s.peer != nil {
					s.peer.close()
				}
			}
		})
		close(h.done)
		if h.listener != nil {
			h.listener.Close()
		}
		if c := h.disc.Load(); c != nil {
			c.Close()
		}
	})
}

func (h *Host) acceptLoop() {
	for {
		conn, err := h.listener.Accept()
		if err != nil {
			return
		}
		go h.handshake(conn)
	}
}

func (h *Host) handshake(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(handshakeTimeout))
	w := newWire(conn, maxHostFrame)
	p, err := w.recv()
	if err != nil || p.Hello == nil {
		conn.Close()
		return
	}
	if subtle.ConstantTimeCompare(p.Hello.Token[:], h.token[:]) != 1 {
		w.send(packet{Reject: "That room code is out of date."})
		conn.Close()
		return
	}
	if p.Hello.Version != ProtocolVersion {
		w.send(packet{Reject: "Your game and the host's are different versions."})
		conn.Close()
		return
	}
	c := newPeer(conn, w)
	var refusal string
	h.call(func() { refusal = h.admit(c, p.Hello.Name) })
	if refusal != "" {
		w.send(packet{Reject: refusal})
		conn.Close()
		return
	}
	conn.SetDeadline(time.Time{})
	go c.writeLoop()
	h.readLoop(c)
}

func (h *Host) admit(c *peer, name string) string {
	switch {
	case h.inMatch():
		return "A match is already in progress."
	case h.occupied() >= MaxPlayers:
		return "The room is full."
	}
	if err := c.w.send(packet{Welcome: true}); err != nil {
		return "connection lost"
	}
	h.seats = append(h.seats, &seat{name: h.uniqueName(cleanName(name)), peer: c})
	h.publishRoom()
	return ""
}

// occupied counts seats that aren't waiting to be cleared.
func (h *Host) occupied() int {
	n := 0
	for _, s := range h.seats {
		if !s.gone {
			n++
		}
	}
	return n
}

// uniqueName numbers a repeated name, so rival records stay separate. A name
// left behind by someone who dropped out is free for them to reclaim.
func (h *Host) uniqueName(name string) string {
	taken := func(candidate string) bool {
		return slices.ContainsFunc(h.seats, func(s *seat) bool { return !s.gone && s.name == candidate })
	}
	candidate := name
	for n := 2; taken(candidate); n++ {
		candidate = fmt.Sprintf("%s %d", name, n)
	}
	return candidate
}

func (h *Host) readLoop(c *peer) {
	defer c.conn.Close()
	var windowStart time.Time
	count := 0
	for {
		p, err := c.w.recv()
		if err != nil {
			h.do(func() { h.drop(c) })
			return
		}
		if p.Action == nil {
			continue
		}
		if t := time.Now(); t.Sub(windowStart) >= time.Second {
			windowStart, count = t, 0
		}
		if count++; count > maxActionsPerSec {
			continue
		}
		a := *p.Action
		h.do(func() { h.remoteAction(c, a) })
	}
}

func (h *Host) seatOf(c *peer) int {
	for i, s := range h.seats {
		if s.peer == c {
			return i
		}
	}
	return -1
}

func (h *Host) remoteAction(c *peer, a match.Action) {
	seat := h.seatOf(c)
	if seat < 0 || h.game == nil {
		return
	}
	// The seat comes from the connection, never from the message.
	a.Player = seat
	h.game.Apply(h.now(), a)
	h.dirty = true
}

func (h *Host) drop(c *peer) {
	index := h.seatOf(c)
	if index < 0 {
		return
	}
	s := h.seats[index]
	s.peer.close()
	s.peer = nil
	if h.inMatch() {
		// Match seats are positional, so the seat stays until the next start.
		h.game.Leave(h.now(), index)
		s.gone = true
		h.dirty = true
	} else {
		h.vacate(index)
	}
	h.publishRoom()
}

// peer is a connected remote player. Its writer always sends the newest
// state, so a slow connection skips frames instead of building a backlog.
type peer struct {
	conn net.Conn
	w    *wire

	mu     sync.Mutex
	room   *Room
	view   *match.View
	notify chan struct{}
	closed chan struct{}
	once   sync.Once
}

func newPeer(conn net.Conn, w *wire) *peer {
	return &peer{conn: conn, w: w, notify: make(chan struct{}, 1), closed: make(chan struct{})}
}

func (p *peer) queueRoom(r *Room) {
	p.mu.Lock()
	p.room = r
	p.mu.Unlock()
	p.wake()
}

func (p *peer) queueView(v *match.View) {
	p.mu.Lock()
	p.view = v
	p.mu.Unlock()
	p.wake()
}

func (p *peer) wake() {
	select {
	case p.notify <- struct{}{}:
	default:
	}
}

func (p *peer) reject(reason string) {
	p.w.send(packet{Reject: reason})
	p.close()
}

func (p *peer) close() {
	p.once.Do(func() {
		close(p.closed)
		p.conn.Close()
	})
}

func (p *peer) writeLoop() {
	for {
		select {
		case <-p.notify:
		case <-p.closed:
			return
		}
		p.mu.Lock()
		room, view := p.room, p.view
		p.room, p.view = nil, nil
		p.mu.Unlock()
		if room != nil && p.w.send(packet{Room: room}) != nil {
			p.close()
			return
		}
		if view != nil && p.w.send(packet{View: view}) != nil {
			p.close()
			return
		}
	}
}
