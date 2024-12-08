package tunnel

import (
	"fmt"
	"io"

	"github.com/jhonoryza/reprox/server/tcp"
)

type TCPTunnel struct {
	tunnel
	publicServer tcp.TCPServer
}

func NewTCP(hostname string, conn io.Writer, maxConsLimit int) (*TCPTunnel, error) {
	t := &TCPTunnel{
		tunnel: newTunnel(hostname, conn, maxConsLimit),
	}

	err := t.privateServer.Init(0, "tcp_tunnel_private_server")
	if err != nil {
		return t, fmt.Errorf("error init private server: %w", err)
	}

	err = t.publicServer.Init(0, "tcp_tunnel_public_server")
	if err != nil {
		return t, fmt.Errorf("error init public server: %w", err)
	}

	return t, nil
}

func (t *TCPTunnel) Protocol() string {
	return "tcp"
}

func (t *TCPTunnel) PublicServerPort() uint16 {
	return t.publicServer.Port()
}

func (t *TCPTunnel) Open() {
	go t.publicServer.Start(t.httpConnectionHandler)
	go t.privateServer.Start(t.privateConnectionHandler)
}

func (t *TCPTunnel) Close() {
	t.publicServer.Stop()
	t.tunnel.Close()
}
