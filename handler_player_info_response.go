package portaldf

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/paroxity/portal/socket/packet"
)

// PlayerInfoResponseHandler is responsible for handling the PlayerInfoResponse packet sent by the proxy.
type PlayerInfoResponseHandler struct {
	waitChan chan struct{}
	response *PlayerInfoResponse
	err      error
}

type PlayerInfoResponse struct {
	PlayerUuid    uuid.UUID
	PlayerAddress string
	Status        byte
	XUID          string
}

// Handle ...
func (h *PlayerInfoResponseHandler) Handle(p packet.Packet, _ *Portal) error {
	pk := p.(*packet.PlayerInfoResponse)
	switch pk.Status {
	case packet.PlayerInfoResponseSuccess:
		h.err = nil
	case packet.PlayerInfoResponsePlayerNotFound:
		h.err = fmt.Errorf("player not found")
	}
	h.response = &PlayerInfoResponse{
		PlayerUuid:    pk.PlayerUUID,
		PlayerAddress: pk.Address,
		Status:        pk.Status,
		XUID:          pk.XUID,
	}
	if h.waitChan != nil {
		close(h.waitChan)
		h.waitChan = nil
	}
	return nil
}
