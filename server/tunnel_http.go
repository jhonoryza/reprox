package main

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/jhonoryza/reprox/package/events"
	"github.com/jhonoryza/reprox/package/utils"
)

type HTTPTunnel struct {
	hostname      string // subdomain.domain
	connPrivate   io.Writer
	mutex         sync.Mutex
	tcpPrivate    TCPServer
	connPublic    map[uint16]net.Conn // port and conn
	initialBuffer map[uint16][]byte   // port and data
}

const DefaultHttpPort = 80

func NewHTTP(hostname string, conn io.Writer) (*HTTPTunnel, error) {
	t := &HTTPTunnel{
		hostname:      hostname,
		connPrivate:   conn,
		connPublic:    make(map[uint16]net.Conn),
		initialBuffer: make(map[uint16][]byte),
	}

	err := t.tcpPrivate.Listen(0, "http-tunnel-private-server")
	if err != nil {
		return t, err
	}
	return t, err
}

func (t *HTTPTunnel) Open() {
	go t.tcpPrivate.ListenerAccept(t.privateHandler)
}

func (t *HTTPTunnel) Close() {
	t.tcpPrivate.ListenerStop()
	for port, conn := range t.connPublic {
		conn.Close()
		delete(t.connPublic, port)
		delete(t.initialBuffer, port)
	}
}

func (t *HTTPTunnel) privateHandler(conn net.Conn) error {
	defer conn.Close()

	// resolve public connection
	buffer := make([]byte, 2)
	_, err := conn.Read(buffer)
	if err != nil {
		return err
	}

	port := binary.LittleEndian.Uint16(buffer)
	publicCon, found := t.connPublic[port]
	if !found {
		return errors.New("public connection not found, cannot pair")
	}
	defer publicCon.Close()

	delete(t.connPublic, port)
	defer delete(t.initialBuffer, port)

	// flush initial buffer to private connection
	if len(t.initialBuffer[port]) > 0 {
		_, err = conn.Write(t.initialBuffer[port])
		if err != nil {
			return err
		}
	}

	// bind public to private connection
	go utils.Bind(publicCon, conn, nil)

	// bind private to public connection
	utils.Bind(conn, publicCon, nil)

	return nil
}

func (t *HTTPTunnel) publicHandler(conn net.Conn, buffer []byte) error {
	ip := conn.RemoteAddr().(*net.TCPAddr).IP
	port := uint16(conn.RemoteAddr().(*net.TCPAddr).Port)
	t.initialBuffer[port] = buffer

	t.mutex.Lock()
	defer t.mutex.Unlock()

	event := events.Event[events.ConnectionReceived]{
		Data: &events.ConnectionReceived{
			ClientIP:    ip,
			ClientPort:  port,
			RateLimited: false,
		},
	}
	err := event.Write(t.connPrivate)
	if err != nil {
		return conn.Close()
	}
	t.connPublic[port] = conn
	return nil
}

func (t *HTTPTunnel) Protocol() string {
	return "http"
}

func (t *HTTPTunnel) Hostname() string {
	return t.hostname
}

func (t *HTTPTunnel) PublicServerPort() uint16 {
	return DefaultHttpPort
}

func (t *HTTPTunnel) PrivateServerPort() uint16 {
	return t.tcpPrivate.Port()
}
