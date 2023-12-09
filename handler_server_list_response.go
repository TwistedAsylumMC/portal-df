package portaldf

import (
	"github.com/paroxity/portal/socket/packet"
)

// ServerListResponseHandler is responsible for handling the ServerListResponse packet sent by the proxy.
type ServerListResponseHandler struct {
	waitChan chan struct{}
	servers  []packet.ServerEntry
}

// Handle ...
func (h *ServerListResponseHandler) Handle(p packet.Packet, _ *Portal) error {
	pk := p.(*packet.ServerListResponse)
	h.servers = pk.Servers
	if h.waitChan != nil {
		close(h.waitChan)
		h.waitChan = nil
	}
	return nil
}
