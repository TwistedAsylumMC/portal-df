package portaldf

import (
	"github.com/paroxity/portal/socket/packet"
	"github.com/patrickmn/go-cache"
	"time"
)

// UpdatePlayerLatencyHandler is responsible for handling the UpdatePlayerLatency packet sent by the proxy.
type UpdatePlayerLatencyHandler struct {
	cache *cache.Cache
}

func NewUpdatePlayerLatencyHandler(defaultExpiration, cleanupInterval time.Duration) *UpdatePlayerLatencyHandler {
	return &UpdatePlayerLatencyHandler{cache: cache.New(defaultExpiration, cleanupInterval)}
}

// Handle ...
func (h *UpdatePlayerLatencyHandler) Handle(p packet.Packet, _ *Portal) error {
	pk := p.(*packet.UpdatePlayerLatency)
	h.cache.Set(pk.PlayerUUID.String(), pk.Latency, cache.DefaultExpiration)
	return nil
}
