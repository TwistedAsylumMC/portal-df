package portaldf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/paroxity/portal/socket/packet"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"go.uber.org/atomic"
	"net"
	"strings"
	"sync"
)

// Portal represents a client that connections to a portal proxy over a TCP socket connection.
type Portal struct {
	address string
	conn    net.Conn

	pool packet.Pool

	sendMu sync.Mutex
	hdr    *packet.Header
	buf    *bytes.Buffer

	name          atomic.String
	authenticated atomic.Bool

	handlers map[uint16]PacketHandler

	authResponseHandler       *AuthResponseHandler
	serverListResponseHandler *ServerListResponseHandler
}

// New returns a new portal client and attempts to connect to the proxy's socket connection on the provided added.
func New(address string) (*Portal, error) {
	portal := &Portal{
		address: address,

		pool: packet.NewPool(),
		buf:  bytes.NewBuffer(make([]byte, 0, 4096)),
		hdr:  &packet.Header{},

		handlers: make(map[uint16]PacketHandler),
	}
	err := portal.connect()
	if err != nil {
		return nil, err
	}
	portal.registerHandlers()
	return portal, nil
}

// connect attempts to connect to the address over a TCP connection.
func (portal *Portal) connect() error {
	conn, err := net.Dial("tcp", portal.address)
	if err != nil {
		return nil
	}
	portal.conn = conn
	portal.name.Store("")
	portal.authenticated.Store(false)
	return nil
}

// registerHandlers registers the default packet handlers for the client.
func (portal *Portal) registerHandlers() {
	portal.authResponseHandler = &AuthResponseHandler{}
	portal.serverListResponseHandler = &ServerListResponseHandler{}

	portal.RegisterPacketHandler(packet.IDAuthResponse, portal.authResponseHandler)
	portal.RegisterPacketHandler(packet.IDServerListResponse, portal.serverListResponseHandler)
}

// RegisterPacketHandler registers the provided packet handler for the provided packet ID.
func (portal *Portal) RegisterPacketHandler(id uint16, handler PacketHandler) {
	portal.handlers[id] = handler
}

// Name returns the name of the client if it is connected to the proxy's socket server. If not connected, an empty
// string will be returned.
func (portal *Portal) Name() string {
	return portal.name.Load()
}

// Authenticate attempts to authenticate the client with the proxy's socket server. The method blocks until it receives
// an auth response from the proxy.
func (portal *Portal) Authenticate(name, secret string) error {
	if portal.authenticated.Load() {
		return fmt.Errorf("attempted to authenticate when already authenticated")
	}
	err := portal.WritePacket(&packet.AuthRequest{
		Protocol: packet.ProtocolVersion,
		Secret:   secret,
		Name:     name,
	})
	if err != nil {
		return err
	}
	c := make(chan *packet.AuthResponse, 1)
	portal.authResponseHandler.responseChan = c
	select {
	case response := <-c:
		if response.Status == packet.AuthResponseSuccess {
			portal.name.Store(name)
			portal.authenticated.Store(true)
			return nil
		}
		switch response.Status {
		case packet.AuthResponseUnsupportedProtocol:
			return fmt.Errorf("attempted to authenticate with an unsupported protocol version. client is using protocol version %d, but the server wants %d", packet.ProtocolVersion, response.Protocol)
		case packet.AuthResponseIncorrectSecret:
			return fmt.Errorf("attempted to connect with an incorrect secret")
		case packet.AuthResponseAlreadyConnected:
			return fmt.Errorf("attempted to connect with a name that is already being used")
		}
		return fmt.Errorf("unknown auth response status %d", response.Status)
	}
}

// Authenticated returns if the client is authenticated with the proxy's socket server.
func (portal *Portal) Authenticated() bool {
	return portal.authenticated.Load()
}

// RegisterServerInfo registers the provided address to the proxy to make it a joinable server.
func (portal *Portal) RegisterServerInfo(address string) error {
	return portal.WritePacket(&packet.RegisterServer{
		Address: address,
	})
}

// Run is a blocking function that constantly attempts to read packets from the client's socket connection.
func (portal *Portal) Run() error {
	if portal.conn == nil {
		return fmt.Errorf("attempting to run without a socket connection")
	}
	for {
		pk, err := portal.readPacket()
		if err != nil {
			if containsAny(err.Error(), "EOF", "closed") {
				return nil
			}
			fmt.Printf("failed to read packet from socket connection: %v\n", err)
			continue
		}

		h, ok := portal.handlers[pk.ID()]
		if ok {
			if err := h.Handle(pk, portal); err != nil {
				fmt.Printf("failed to handle packet from socket connection: %v\n", err)
			}
		} else {
			fmt.Printf("unhandled packet %T from socket connection: %v\n", pk, err)
		}
	}
}

// Transfer sends a transfer request to portal, requesting to transfer the player to the server with the provided name.
func (portal *Portal) Transfer(p *player.Player, server string) error {
	return portal.WritePacket(&packet.TransferRequest{
		PlayerUUID: p.UUID(),
		Server:     server,
	})
}

// TODO: PlayerInfo returns the information of the player provided. This information includes their IP address and their XUID.

// ServerList returns a list of servers that are registered to the connected portal.
func (portal *Portal) ServerList() ([]packet.ServerEntry, error) {
	handler := portal.serverListResponseHandler
	if handler.waitChan != nil {
		err := portal.WritePacket(&packet.ServerListRequest{})
		if err != nil {
			return nil, err
		}
		handler.waitChan = make(chan struct{}, 1)
	}
	select {
	case <-handler.waitChan:
		return handler.servers, nil
	}
}

// TODO: FindPlayer returns the name of the server the player is connected to. If they are not connected to portal, an empty
// string and false will be returned.

// TODO: PlayerLatency returns the latency of the player provided.

// ReadPacket reads a packet from the connection and returns it. The client is expected to prefix the packet payload
// with 4 bytes for the length of the payload.
func (portal *Portal) readPacket() (pk packet.Packet, err error) {
	var l uint32
	if err := binary.Read(portal.conn, binary.LittleEndian, &l); err != nil {
		return nil, err
	}

	data := make([]byte, l)
	read, err := portal.conn.Read(data)
	if err != nil {
		return nil, err
	}
	if read != int(l) {
		return nil, fmt.Errorf("expected %v bytes, got %v", l, read)
	}

	buf := bytes.NewBuffer(data)
	header := &packet.Header{}
	if err := header.Read(buf); err != nil {
		return nil, err
	}

	pk, ok := portal.pool[header.PacketID]
	if !ok {
		return nil, fmt.Errorf("unknown packet %v", header.PacketID)
	}

	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = fmt.Errorf("%T: %w", pk, recoveredErr.(error))
		}
	}()
	pk.Unmarshal(protocol.NewReader(buf, 0, true))
	if buf.Len() > 0 {
		return nil, fmt.Errorf("still have %v bytes unread", buf.Len())
	}

	return pk, nil
}

// WritePacket writes a packet to the client. Since it's a TCP connection, the payload is prefixed with a length so the
// client can read the exact length of the packet.
func (portal *Portal) WritePacket(pk packet.Packet) error {
	portal.sendMu.Lock()
	portal.hdr.PacketID = pk.ID()
	_ = portal.hdr.Write(portal.buf)

	pk.Marshal(protocol.NewWriter(portal.buf, 0))

	data := portal.buf.Bytes()
	portal.buf.Reset()
	portal.sendMu.Unlock()

	buf := bytes.NewBuffer(make([]byte, 0, 4+len(data)))

	if err := binary.Write(buf, binary.LittleEndian, int32(len(data))); err != nil {
		return err
	}
	if _, err := buf.Write(data); err != nil {
		return err
	}

	if _, err := portal.conn.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

// containsAny checks if the string contains any of the provided sub strings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
