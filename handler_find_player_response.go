package portaldf

import (
	"fmt"
	"github.com/paroxity/portal/socket/packet"
)

// FindPlayerResponseHandler is responsible for handling the FindPlayer packet sent by the proxy.
type FindPlayerResponseHandler struct {
	waitChan chan struct{}
	response string
	err      error
}

// Handle ...
func (h *FindPlayerResponseHandler) Handle(p packet.Packet, _ *Portal) error {
	pk := p.(*packet.FindPlayerResponse)
	if pk.Online {
		h.err = nil
		h.response = pk.Server
	} else {
		h.err = fmt.Errorf("player not online")
	}
	if h.waitChan != nil {
		close(h.waitChan)
		h.waitChan = nil
	}
	return nil
}
