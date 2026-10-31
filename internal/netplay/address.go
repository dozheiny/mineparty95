package netplay

import (
	"net"
)

// CodeInfo is a room code and the kind of network it works on.
type CodeInfo struct {
	Label string
	Code  string
}

func (h *Host) codeFor(ip net.IP) string {
	var a Address
	copy(a.IP[:], ip.To4())
	a.Port = uint16(h.port)
	return EncodeCode(a, h.token)
}

// LocalCode returns a code that works on this machine's main network.
func (h *Host) LocalCode() string {
	if h.opts.Loopback {
		return h.codeFor(net.IPv4(127, 0, 0, 1))
	}
	if ip := outboundIP(nil); ip != nil {
		return h.codeFor(ip)
	}
	return h.codeFor(net.IPv4(127, 0, 0, 1))
}

// codeFacing returns the code a machine at peer should use to reach us.
func (h *Host) codeFacing(peer net.IP) string {
	if h.opts.Loopback {
		return h.LocalCode()
	}
	if ip := outboundIP(peer); ip != nil {
		return h.codeFor(ip)
	}
	return h.LocalCode()
}

// outboundIP asks the OS which local address routes towards dest. Dialling
// UDP sends nothing; it only makes the kernel choose. A nil dest means the
// default route.
func outboundIP(dest net.IP) net.IP {
	target := "192.0.2.1:9" // reserved documentation address
	if dest != nil {
		target = net.JoinHostPort(dest.String(), "9")
	}
	conn, err := net.Dial("udp4", target)
	if err != nil {
		return nil
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP
}

// Codes lists every room code worth sharing, the most widely usable first.
func (h *Host) Codes() []CodeInfo {
	var out []CodeInfo
	if ext := h.internetCode(); ext != "" {
		out = append(out, CodeInfo{Label: "Internet", Code: ext})
	}
	main := h.LocalCode()
	out = append(out, CodeInfo{Label: "LAN", Code: main})
	if h.opts.Loopback {
		return out
	}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok || ipn.IP.To4() == nil || !isOverlay(ipn.IP) {
			continue
		}
		if code := h.codeFor(ipn.IP); code != main {
			out = append(out, CodeInfo{Label: "VPN", Code: code})
		}
	}
	return out
}

// isOverlay matches the carrier-grade NAT range used by Tailscale and the
// 10.147.x.x style ranges are left to the LAN entry.
func isOverlay(ip net.IP) bool {
	v4 := ip.To4()
	return v4 != nil && v4[0] == 100 && v4[1]&0xC0 == 64
}
