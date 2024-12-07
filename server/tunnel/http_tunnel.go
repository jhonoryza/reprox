package tunnel

import (
	"io"
	"net"
)

const DefaultHttpPort = 80

type HTTPTunnel struct {
	tunnel
}

func NewHTTP(hostname string, conn io.Writer, maxConsLimit int) (*HTTPTunnel, error) {
	t := &HTTPTunnel{
		tunnel: newTunnel(hostname, conn, maxConsLimit),
	}

	// Membuka listener TCP pada port random
	err := t.privateServer.Init(0, "http-tunnel-private-server")
	if err != nil {
		return t, err
	}
	return t, err
}

func (t *HTTPTunnel) Open() {
	go t.privateServer.Start(t.privateConnectionHandler)
}

func (t *HTTPTunnel) Protocol() string {
	return "http"
}

func (t *HTTPTunnel) PublicServerPort() uint16 {
	return DefaultHttpPort
}

func (t *HTTPTunnel) HttpConnectionHandler(conn net.Conn, buffer []byte) error {
	port := uint16(conn.RemoteAddr().(*net.TCPAddr).Port)

	t.initialBuffer[port] = buffer
	return t.httpConnectionHandler(conn)
}
