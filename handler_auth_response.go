package portaldf

import (
	"github.com/paroxity/portal/socket/packet"
)

// AuthResponseHandler is responsible for handling the AuthResponse packet sent by the proxy.
type AuthResponseHandler struct {
	responseChan chan *packet.AuthResponse
}

// Handle ...
func (h *AuthResponseHandler) Handle(p packet.Packet, _ *Portal) error {
	if h.responseChan != nil {
		h.responseChan <- p.(*packet.AuthResponse)
		h.responseChan = nil
	}
	return nil
}
