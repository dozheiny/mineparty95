package netplay

import (
	"strings"
	"testing"
	"time"

	"mineparty/internal/engine"
	"mineparty/internal/match"
)

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	waitUpTo(t, 10*time.Second, what, cond)
}

func waitLong(t *testing.T, what string, cond func() bool) {
	t.Helper()
	waitUpTo(t, 90*time.Second, what, cond)
}

func waitUpTo(t *testing.T, limit time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestRoomCodeRoundTrips(t *testing.T) {
	addr := Address{IP: [4]byte{203, 0, 113, 77}, Port: 37195}
	code := EncodeCode(addr, [4]byte{1, 2, 3, 4})
	if len(strings.ReplaceAll(code, "-", "")) != 16 {
		t.Fatalf("code %q should have 16 symbols", code)
	}
	got, token, err := DecodeCode(code)
	if err != nil || got != addr || token != [4]byte{1, 2, 3, 4} {
		t.Fatalf("got %+v %v err=%v", got, token, err)
	}
}

func TestRoomCodeForgivesTypingMistakes(t *testing.T) {
	addr := Address{IP: [4]byte{10, 1, 2, 3}, Port: 40000}
	code := EncodeCode(addr, [4]byte{9, 9, 9, 9})
	messy := " " + strings.ToLower(strings.ReplaceAll(code, "-", " ")) + "\n"
	messy = strings.NewReplacer("0", "o", "1", "l").Replace(messy)
	if got, _, err := DecodeCode(messy); err != nil || got != addr {
		t.Fatalf("got %+v err=%v", got, err)
	}
}

func TestRoomCodeRejectsGarbage(t *testing.T) {
	for _, s := range []string{"", "ABC", "!!!!-!!!!-!!!!-!!!!", strings.Repeat("A", 40)} {
		if _, _, err := DecodeCode(s); err == nil {
			t.Fatalf("%q decoded without error", s)
		}
	}
}

func newHost(t *testing.T, name string) *Host {
	t.Helper()
	h, err := NewHost(HostOptions{Name: name, Port: 0, Loopback: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	return h
}

func join(t *testing.T, h *Host, name string) *Client {
	t.Helper()
	c, err := Join(h.LocalCode(), name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestClientJoinsRoomAndAppearsInLobby(t *testing.T) {
	h := newHost(t, "alice")
	c := join(t, h, "bob")
	waitFor(t, "both players listed", func() bool {
		return len(h.Room().Players) == 2 && len(c.Room().Players) == 2
	})
	if got := c.Room().Players[1].Name; got != "bob" {
		t.Fatalf("name = %q", got)
	}
	if c.You() != 1 {
		t.Fatalf("You = %d", c.You())
	}
}

func TestWrongTokenIsRefused(t *testing.T) {
	h := newHost(t, "alice")
	addr, token, _ := DecodeCode(h.LocalCode())
	token[0] ^= 0xff
	if _, err := Join(EncodeCode(addr, token), "mallory"); err == nil {
		t.Fatal("a wrong token must not be admitted")
	}
	if n := len(h.Room().Players); n != 1 {
		t.Fatalf("players = %d", n)
	}
}

func TestRoomRefusesFifthPlayer(t *testing.T) {
	h := newHost(t, "a")
	join(t, h, "b")
	join(t, h, "c")
	join(t, h, "d")
	if _, err := Join(h.LocalCode(), "e"); err == nil {
		t.Fatal("the fifth player must be refused")
	}
}

func TestJoinDuringMatchIsRefused(t *testing.T) {
	h := newHost(t, "a")
	join(t, h, "b")
	waitFor(t, "lobby", func() bool { return len(h.Room().Players) == 2 })
	h.Configure(match.Race, engine.Beginner)
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "match start", func() bool { return h.Room().InMatch })
	if _, err := Join(h.LocalCode(), "late"); err == nil {
		t.Fatal("a latecomer must be refused")
	}
}

func TestStartEnforcesModePlayerLimits(t *testing.T) {
	h := newHost(t, "a")
	h.Configure(match.Race, engine.Beginner)
	if err := h.Start(); err == nil {
		t.Fatal("race needs two players")
	}
	join(t, h, "b")
	join(t, h, "c")
	waitFor(t, "lobby", func() bool { return len(h.Room().Players) == 3 })
	h.Configure(match.Flags, engine.Beginner)
	if err := h.Start(); err == nil {
		t.Fatal("flags is a duel; three players must be refused")
	}
}

func TestClientActionsReachMatchAndViewsReturn(t *testing.T) {
	h := newHost(t, "a")
	c := join(t, h, "b")
	waitFor(t, "lobby", func() bool { return len(h.Room().Players) == 2 })
	h.Configure(match.Coop, engine.Beginner)
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "playing", func() bool {
		v := c.View()
		return v != nil && v.Phase == match.Playing
	})
	v := c.View()
	var target = -1
	for i, tile := range v.Tiles {
		if tile == match.TileHidden {
			target = i
			break
		}
	}
	c.Send(match.Action{Kind: match.Flag, Tile: target})
	waitFor(t, "flag visible on the host", func() bool {
		hv := h.View()
		return hv != nil && hv.Tiles[target] == match.TileFlag
	})
	if c.You() != 1 {
		t.Fatalf("client identity = %d", c.You())
	}
}

func TestClientCannotActAsAnotherPlayer(t *testing.T) {
	h := newHost(t, "a")
	c := join(t, h, "b")
	waitFor(t, "lobby", func() bool { return len(h.Room().Players) == 2 })
	h.Configure(match.Coop, engine.Beginner)
	h.Start()
	waitFor(t, "playing", func() bool {
		v := h.View()
		return v != nil && v.Phase == match.Playing
	})
	c.Send(match.Action{Player: 0, Kind: match.Cursor, Tile: 7})
	waitFor(t, "cursor update", func() bool { return h.View().Players[1].Cursor == 7 })
	if got := h.View().Players[0].Cursor; got == 7 {
		t.Fatal("the host trusted a forged player id")
	}
}

func TestHostClosingEndsClientSession(t *testing.T) {
	h := newHost(t, "a")
	c := join(t, h, "b")
	waitFor(t, "lobby", func() bool { return len(c.Room().Players) == 2 })
	h.Close()
	waitFor(t, "client error", func() bool { return c.Err() != nil })
}

func TestClientLeavingInLobbyFreesSlot(t *testing.T) {
	h := newHost(t, "a")
	c := join(t, h, "b")
	waitFor(t, "joined", func() bool { return len(h.Room().Players) == 2 })
	c.Close()
	waitFor(t, "left", func() bool { return len(h.Room().Players) == 1 })
}

func TestClientLeavingDuelGivesHostTheWin(t *testing.T) {
	h := newHost(t, "a")
	c := join(t, h, "b")
	waitFor(t, "lobby", func() bool { return len(h.Room().Players) == 2 })
	h.Configure(match.Flags, engine.Beginner)
	h.Start()
	waitFor(t, "match", func() bool { return h.Room().InMatch })
	c.Close()
	waitFor(t, "forfeit", func() bool {
		v := h.View()
		return v != nil && v.Phase == match.Over && v.Winner == 0
	})
}

// finishRaceWithBot plays a race to its end, won by a fast CPU in seat 2.
func finishRaceWithBot(t *testing.T, h *Host) {
	t.Helper()
	h.AddBot()
	waitFor(t, "seats", func() bool { return len(h.Room().Players) == 3 })
	h.SetBotDelay(time.Millisecond)
	h.Configure(match.Race, engine.Beginner)
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "match over", func() bool {
		v := h.View()
		return v != nil && v.Phase == match.Over
	})
}

func TestLeavingBetweenMatchesFreesTheSeatForRejoining(t *testing.T) {
	h := newHost(t, "a")
	c := join(t, h, "b")
	waitFor(t, "joined", func() bool { return len(h.Room().Players) == 2 })
	finishRaceWithBot(t, h)
	c.Close()
	waitFor(t, "seat freed", func() bool { return len(h.Room().Players) == 2 })
	again := join(t, h, "b")
	waitFor(t, "rejoined", func() bool { return len(again.Room().Players) == 3 })
	if got := again.Room().Players[2].Name; got != "b" {
		t.Fatalf("rejoining renamed the player to %q", got)
	}
}

func TestActionsAfterRosterChangeCannotReachTheFinishedMatch(t *testing.T) {
	h := newHost(t, "a")
	c := join(t, h, "b")
	stay := join(t, h, "c")
	waitFor(t, "joined", func() bool { return len(h.Room().Players) == 3 })
	h.Configure(match.Coop, engine.Beginner)
	h.AddBot()
	waitFor(t, "seats", func() bool { return len(h.Room().Players) == 4 })
	h.SetBotDelay(time.Millisecond)
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "over", func() bool {
		v := h.View()
		return v != nil && v.Phase == match.Over
	})
	c.Close()
	waitFor(t, "seat freed", func() bool { return len(h.Room().Players) == 3 })
	stay.Send(match.Action{Kind: match.Cursor, Tile: 1})
	if err := h.Start(); err != nil {
		t.Fatalf("the room should still be able to rematch: %v", err)
	}
}

func TestRematchStartsAFreshBoardAndTracksWins(t *testing.T) {
	h := newHost(t, "a")
	h.AddBot()
	waitFor(t, "lobby", func() bool { return len(h.Room().Players) == 2 })
	h.SetBotDelay(time.Millisecond)
	h.Configure(match.Race, engine.Beginner)
	h.Start()
	waitFor(t, "bot finishes", func() bool {
		v := h.View()
		return v != nil && v.Phase == match.Over
	})
	if got := h.Room().Players[1].Wins; got != 1 {
		t.Fatalf("bot wins = %d", got)
	}
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "new match", func() bool {
		v := h.View()
		return v != nil && v.Phase != match.Over
	})
}

func TestKickRemovesAPlayerFromTheLobby(t *testing.T) {
	h := newHost(t, "a")
	h.AddBot()
	waitFor(t, "bot", func() bool { return len(h.Room().Players) == 2 })
	h.Kick(1)
	waitFor(t, "kick", func() bool { return len(h.Room().Players) == 1 })
}

func TestEndToEndAllModesWithBotsAndARemoteClient(t *testing.T) {
	for _, tc := range []struct {
		mode match.Mode
		bots int
	}{{match.Race, 1}, {match.Coop, 0}, {match.Flags, 0}, {match.Territory, 2}} {
		t.Run(tc.mode.String(), func(t *testing.T) {
			h := newHost(t, "a")
			h.SetBotDelay(time.Millisecond)
			c := join(t, h, "b")
			for range tc.bots {
				h.AddBot()
			}
			waitFor(t, "room", func() bool { return len(c.Room().Players) == 2+tc.bots })
			h.Configure(tc.mode, engine.Beginner)
			if err := h.Start(); err != nil {
				t.Fatal(err)
			}
			// Bots stand in for both humans, through the public API only.
			hostBot, clientBot := match.NewBot(0, 91), match.NewBot(1, 92)
			waitLong(t, "match over", func() bool {
				if v := h.View(); v != nil {
					if a, ok := hostBot.Next(*v); ok {
						h.Send(a)
					}
				}
				if v := c.View(); v != nil {
					if a, ok := clientBot.Next(*v); ok {
						c.Send(a)
					}
				}
				v := c.View()
				return v != nil && v.Phase == match.Over
			})
		})
	}
}

func TestManualPublicIPProducesAnInternetCode(t *testing.T) {
	h := newHost(t, "a")
	if err := h.SetPublicIP("203.0.113.9"); err != nil {
		t.Fatal(err)
	}
	codes := h.Codes()
	if codes[0].Label != "Internet" {
		t.Fatalf("codes = %+v", codes)
	}
	addr, _, err := DecodeCode(codes[0].Code)
	if err != nil || addr.IP != [4]byte{203, 0, 113, 9} {
		t.Fatalf("addr = %+v err=%v", addr, err)
	}
	local, _, _ := DecodeCode(h.LocalCode())
	if addr.Port != local.Port {
		t.Fatalf("port %d should match the listening port %d", addr.Port, local.Port)
	}
}

func TestManualPublicIPRejectsNonsense(t *testing.T) {
	h := newHost(t, "a")
	for _, ip := range []string{"", "hello", "999.1.1.1", "::1"} {
		if err := h.SetPublicIP(ip); err == nil {
			t.Fatalf("%q was accepted", ip)
		}
	}
}

func TestLANDiscoveryFindsHost(t *testing.T) {
	h := newHost(t, "alice")
	stop, err := h.ServeDiscovery(0)
	if err != nil {
		t.Skipf("discovery socket unavailable: %v", err)
	}
	defer stop()
	games := Discover(h.DiscoveryPort(), 500*time.Millisecond)
	if len(games) != 1 || games[0].Name != "alice" || games[0].Code == "" {
		t.Fatalf("games = %+v", games)
	}
}

func TestDiscoveryListsARoomOnceHoweverManyRoutesAnswer(t *testing.T) {
	token := [4]byte{1, 2, 3, 4}
	viaLAN := EncodeCode(Address{IP: [4]byte{192, 168, 1, 5}, Port: 9000}, token)
	viaLoop := EncodeCode(Address{IP: [4]byte{127, 0, 0, 1}, Port: 9000}, token)
	other := EncodeCode(Address{IP: [4]byte{192, 168, 1, 6}, Port: 9000}, [4]byte{9, 9, 9, 9})
	games := uniqueRooms([]LANGame{{Name: "a", Code: viaLoop}, {Name: "a", Code: viaLAN}, {Name: "b", Code: other}})
	if len(games) != 2 {
		t.Fatalf("games = %+v", games)
	}
	if games[0].Code != viaLAN {
		t.Fatalf("prefer the routable address, got %+v", games[0])
	}
}

func TestDuplicateNamesAreMadeUnique(t *testing.T) {
	h := newHost(t, "sam")
	join(t, h, "sam")
	join(t, h, "sam")
	waitFor(t, "joined", func() bool { return len(h.Room().Players) == 3 })
	seen := map[string]bool{}
	for _, p := range h.Room().Players {
		if seen[p.Name] {
			t.Fatalf("duplicate name in %+v", h.Room().Players)
		}
		seen[p.Name] = true
	}
}

func TestHandshakeRejectsOversizedFrames(t *testing.T) {
	h := newHost(t, "a")
	if err := sendOversizedHello(h.LocalCode()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if n := len(h.Room().Players); n != 1 {
		t.Fatalf("players = %d", n)
	}
}
