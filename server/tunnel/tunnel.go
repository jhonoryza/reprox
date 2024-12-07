package tunnel

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/jhonoryza/reprox/server/events"
	"github.com/jhonoryza/reprox/server/tcp"
)

type tunnel struct {
	hostname      string
	maxConsLimit  int
	eventWriter   io.Writer
	privateServer tcp.TCPServer
	publicCons    map[uint16]net.Conn // isinya port dan conn
	initialBuffer map[uint16][]byte   // isinya port dan data
	eventWriterMx sync.Mutex
}

func newTunnel(hostname string, conn io.Writer, maxConsLimit int) tunnel {
	return tunnel{
		hostname:      hostname,
		maxConsLimit:  maxConsLimit,
		eventWriter:   conn,
		publicCons:    make(map[uint16]net.Conn),
		initialBuffer: make(map[uint16][]byte),
	}
}

type Tunnel interface {
	Open()
	Close()

	Hostname() string
	Protocol() string
	PrivateServerPort() uint16
	PublicServerPort() uint16
}

func (t *tunnel) Close() {
	t.privateServer.Stop()
	for port, conn := range t.publicCons {
		conn.Close()
		delete(t.publicCons, port)
		delete(t.initialBuffer, port)
	}
}

func (t *tunnel) Hostname() string {
	return t.hostname
}

func (t *tunnel) PrivateServerPort() uint16 {
	return t.privateServer.Port()
}

/**
 * resolve public connection from variable publicCons
 * then bind private to public connection and virce versa
 */
func (t *tunnel) privateConnectionHandler(conn net.Conn) error {
	defer conn.Close()

	// resolve public connection
	buffer := make([]byte, 2)
	_, err := conn.Read(buffer)
	if err != nil {
		return err
	}

	port := binary.LittleEndian.Uint16(buffer)
	publicCon, found := t.publicCons[port]
	if !found {
		return errors.New("public connection not found, cannot pair")
	}
	defer publicCon.Close()

	delete(t.publicCons, port)
	defer delete(t.initialBuffer, port)

	// flush initial buffer to private connection
	if len(t.initialBuffer[port]) > 0 {
		_, err = conn.Write(t.initialBuffer[port])
		if err != nil {
			return err
		}
	}

	// bind public to private connection
	go Bind(publicCon, conn, nil)

	// bind private to public connection
	Bind(conn, publicCon, nil)

	return nil
}

/**
 * read data from src connection
 * then send the data to dst connection
 */
func Bind(src net.Conn, dst net.Conn, debug io.Writer) error {
	defer src.Close()
	defer dst.Close()

	buf := make([]byte, 4096)

	for {
		_ = src.SetReadDeadline(time.Now().Add(time.Second))
		n, err := src.Read(buf)
		if err == io.EOF {
			break
		}

		_ = dst.SetWriteDeadline(time.Now().Add(time.Second))
		_, err = dst.Write(buf[:n])
		if err != nil {
			return err
		}
		if debug != nil {
			debug.Write(buf[:n])
		}
		time.Sleep(time.Millisecond * 10)
	}

	return nil
}

func (t *tunnel) httpConnectionHandler(conn net.Conn, port uint16) error {
	ip := conn.RemoteAddr().(*net.TCPAddr).IP

	t.eventWriterMx.Lock()
	defer t.eventWriterMx.Unlock()

	event := events.Event[events.ConnectionReceived]{
		Data: &events.ConnectionReceived{
			ClientIP:    ip,
			ClientPort:  port,
			RateLimited: false,
		},
	}
	err := event.Write(t.eventWriter)
	if err != nil {
		return conn.Close()
	}
	t.publicCons[port] = conn
	return nil
}
