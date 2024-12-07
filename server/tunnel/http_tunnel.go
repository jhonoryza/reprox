package tunnel

import "io"

const DefaultHttpPort = 80

type HttpTunnel struct {
	tunnel
}

func NewHTTP(hostname string, conn io.Writer, maxConsLimit int) (*HttpTunnel, error) {
	t := &HttpTunnel{
		tunnel: newTunnel(hostname, conn, maxConsLimit),
	}

	// Membuka listener TCP pada port random
	err := t.privateServer.Init(0, "http-tunnel-private-server")
	if err != nil {
		return t, err
	}
	return t, err
}

func (t *HttpTunnel) Open() {
	go t.privateServer.Start(t.privateConnectionHandler)
}

func (t *HttpTunnel) Protocol() string {
	return "http"
}

func (t *HttpTunnel) PublicServerPort() uint16 {
	return DefaultHttpPort
}
