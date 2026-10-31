package netplay

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway2"
)

const (
	upnpTimeout = 8 * time.Second
	upnpLease   = 4 * 60 * 60 // seconds
)

// gateway is the part of the IGD WANIPConnection services we use. Both the
// v1 and v2 clients satisfy it.
type gateway interface {
	AddPortMappingCtx(ctx context.Context, remoteHost string, externalPort uint16, protocol string, internalPort uint16, internalClient string, enabled bool, description string, leaseDuration uint32) error
	DeletePortMappingCtx(ctx context.Context, remoteHost string, externalPort uint16, protocol string) error
	GetExternalIPAddressCtx(ctx context.Context) (string, error)
}

type internetState struct {
	mu      sync.Mutex
	code    string
	status  string
	failed  bool
	gateway gateway
	port    uint16
}

// OpenInternet asks the router to forward our port, returning quickly and
// doing the work in the background. Watch InternetStatus for the outcome.
func (h *Host) OpenInternet() {
	h.setInternet("", "Asking your router to open a port...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), upnpTimeout)
		defer cancel()
		if err := h.mapPort(ctx); err != nil {
			h.setInternet("", fmt.Sprintf("Automatic setup failed. For internet play, forward TCP port %d on your router and enter your public IP below, or use Tailscale.", h.port))
			h.internet.mu.Lock()
			h.internet.failed = true
			h.internet.mu.Unlock()
		}
	}()
}

// SetPublicIP makes an internet code for a port the player forwarded by
// hand. Nothing here can discover the address without a third-party service.
func (h *Host) SetPublicIP(text string) error {
	ip := net.ParseIP(strings.TrimSpace(text)).To4()
	if ip == nil {
		return errors.New("that doesn't look like an IPv4 address")
	}
	h.setInternet(h.codeFor(ip), fmt.Sprintf("Using your public IP. TCP port %d must be forwarded to this computer.", h.port))
	h.internet.mu.Lock()
	h.internet.failed = false
	h.internet.mu.Unlock()
	return nil
}

// NeedsManualIP reports whether automatic internet setup has given up.
func (h *Host) NeedsManualIP() bool {
	h.internet.mu.Lock()
	defer h.internet.mu.Unlock()
	return h.internet.failed
}

func (h *Host) mapPort(ctx context.Context) error {
	gw, local, err := findGateway(ctx)
	if err != nil {
		return err
	}
	port := uint16(h.port)
	if err := gw.AddPortMappingCtx(ctx, "", port, "TCP", port, local.String(), true, "Mine Party 95", upnpLease); err != nil {
		return err
	}
	h.internet.mu.Lock()
	h.internet.gateway, h.internet.port = gw, port
	h.internet.mu.Unlock()

	ext, err := gw.GetExternalIPAddressCtx(ctx)
	if err != nil {
		return err
	}
	ip := net.ParseIP(ext).To4()
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || isOverlay(ip) {
		// The router itself sits behind another NAT.
		return errors.New("router has no public address")
	}
	h.setInternet(h.codeFor(ip), "Port opened. Internet code ready.")
	return nil
}

func findGateway(ctx context.Context) (gateway, net.IP, error) {
	if clients, _, err := internetgateway2.NewWANIPConnection2ClientsCtx(ctx); err == nil && len(clients) > 0 {
		return clients[0], routerFacingIP(clients[0].ServiceClient.RootDevice.URLBase.Hostname()), nil
	}
	if clients, _, err := internetgateway2.NewWANIPConnection1ClientsCtx(ctx); err == nil && len(clients) > 0 {
		return clients[0], routerFacingIP(clients[0].ServiceClient.RootDevice.URLBase.Hostname()), nil
	}
	if clients, _, err := internetgateway2.NewWANPPPConnection1ClientsCtx(ctx); err == nil && len(clients) > 0 {
		return clients[0], routerFacingIP(clients[0].ServiceClient.RootDevice.URLBase.Hostname()), nil
	}
	return nil, nil, errors.New("no UPnP router found")
}

func routerFacingIP(host string) net.IP {
	return outboundIP(net.ParseIP(host))
}

func (h *Host) setInternet(code, status string) {
	h.internet.mu.Lock()
	defer h.internet.mu.Unlock()
	h.internet.code, h.internet.status = code, status
}

func (h *Host) internetCode() string {
	h.internet.mu.Lock()
	defer h.internet.mu.Unlock()
	return h.internet.code
}

// InternetStatus describes how internet play is doing, for the lobby.
func (h *Host) InternetStatus() string {
	h.internet.mu.Lock()
	defer h.internet.mu.Unlock()
	return h.internet.status
}

// closeInternet removes the router's port forward.
func (h *Host) closeInternet() {
	h.internet.mu.Lock()
	gw, port := h.internet.gateway, h.internet.port
	h.internet.gateway = nil
	h.internet.mu.Unlock()
	if gw == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	gw.DeletePortMappingCtx(ctx, "", port, "TCP")
}
