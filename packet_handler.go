package portaldf

import "github.com/paroxity/portal/socket/packet"

// PacketHandler represents a type which handles a specific packet coming from a client.
type PacketHandler interface {
	// Handle is responsible for handling an incoming packet for the client.
	Handle(p packet.Packet, portal *Portal) error
}
