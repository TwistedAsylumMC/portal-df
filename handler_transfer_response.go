package portaldf

import (
	"fmt"
	"github.com/paroxity/portal/socket/packet"
)

// TransferResponseHandler is responsible for handling the TransferResponse packet sent by the proxy.
type TransferResponseHandler struct {
	waitChan       chan struct{}
	responseStatus byte
	err            error
}

// Handle ...
func (h *TransferResponseHandler) Handle(p packet.Packet, _ *Portal) error {
	pk := p.(*packet.TransferResponse)
	switch pk.Status {
	case packet.TransferResponseSuccess:
		h.err = nil
	case packet.TransferResponsePlayerNotFound:
		h.err = fmt.Errorf("player not found")
	case packet.TransferResponseAlreadyOnServer:
		h.err = fmt.Errorf("already on server")
	case packet.TransferResponseServerNotFound:
		h.err = fmt.Errorf("server not found")
	case packet.TransferResponseError:
		h.err = fmt.Errorf(pk.Error)
	}
	h.responseStatus = pk.Status
	if h.waitChan != nil {
		close(h.waitChan)
		h.waitChan = nil
	}
	return nil
}
