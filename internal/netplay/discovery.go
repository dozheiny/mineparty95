package netplay

import (
	"encoding/json"
	"net"
	"sort"
	"time"
)

// DiscoveryPort is the UDP port hosts answer LAN searches on.
const DiscoveryPort = 37196

const discoveryProbe = "MINEPARTY?"

// LANGame is a room found on the local network.
type LANGame struct {
	Name    string
	Code    string
	Players int
	Version int
}

// ServeDiscovery answers LAN searches until stopped. Port 0 picks a free
// port, for tests. It fails if the port is taken, typically by another copy
// of the game; that host can still be joined by code.
func (h *Host) ServeDiscovery(port int) (stop func(), err error) {
	pc, err := net.ListenUDP("udp4", &net.UDPAddr{Port: port})
	if err != nil {
		return nil, err
	}
	h.discPort.Store(int32(pc.LocalAddr().(*net.UDPAddr).Port))
	h.disc.Store(pc)
	go func() {
		buf := make([]byte, 64)
		for {
			n, from, err := pc.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if string(buf[:n]) != discoveryProbe {
				continue
			}
			room := h.Room()
			if room.InMatch {
				continue
			}
			reply, _ := json.Marshal(LANGame{
				Name:    room.Players[0].Name,
				Code:    h.codeFacing(from.IP),
				Players: len(room.Players),
				Version: ProtocolVersion,
			})
			pc.WriteToUDP(reply, from)
		}
	}()
	return func() { pc.Close() }, nil
}

// DiscoveryPort returns the port ServeDiscovery bound.
func (h *Host) DiscoveryPort() int { return int(h.discPort.Load()) }

// Discover searches the local network for rooms.
func Discover(port int, wait time.Duration) []LANGame {
	pc, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil
	}
	defer pc.Close()

	targets := []net.IP{net.IPv4bcast, net.IPv4(127, 0, 0, 1)}
	targets = append(targets, broadcastAddrs()...)
	for _, ip := range targets {
		pc.WriteToUDP([]byte(discoveryProbe), &net.UDPAddr{IP: ip, Port: port})
	}

	var found []LANGame
	buf := make([]byte, 512)
	pc.SetReadDeadline(time.Now().Add(wait))
	for {
		n, _, err := pc.ReadFromUDP(buf)
		if err != nil {
			break
		}
		var g LANGame
		if json.Unmarshal(buf[:n], &g) == nil && g.Code != "" && g.Version == ProtocolVersion {
			found = append(found, g)
		}
	}
	return uniqueRooms(found)
}

// uniqueRooms merges replies from one room. A host answers on every network
// it shares with the searcher, each with its own address but the same token.
func uniqueRooms(games []LANGame) []LANGame {
	byToken := map[[4]byte]LANGame{}
	for _, g := range games {
		addr, token, err := DecodeCode(g.Code)
		if err != nil {
			continue
		}
		have, seen := byToken[token]
		if seen {
			if known, _, _ := DecodeCode(have.Code); known.IP[0] != 127 || addr.IP[0] == 127 {
				continue
			}
		}
		byToken[token] = g
	}
	out := make([]LANGame, 0, len(byToken))
	for _, g := range byToken {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Code < out[j].Code
	})
	return out
}

func broadcastAddrs() []net.IP {
	var out []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagBroadcast == 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok || ipn.IP.To4() == nil {
				continue
			}
			ip := ipn.IP.To4()
			b := make(net.IP, 4)
			for i := range b {
				b[i] = ip[i] | ^ipn.Mask[len(ipn.Mask)-4+i]
			}
			out = append(out, b)
		}
	}
	return out
}
