// Package ws provides a net.Conn implementation over a WebSocket connection,
// allowing the existing gob-based event protocol and raw byte piping to run
// unchanged on top of a WebSocket transport.
//
// Every Write call is sent as a single binary WebSocket message. Reads are
// stream-oriented: message boundaries are hidden behind an internal buffer so
// callers can do arbitrary-sized reads like they would on a TCP connection.
package ws

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket transport endpoint paths, shared with the server.
const (
	EventPath = "/_reprox/event"
	DataPath  = "/_reprox/data"
)

// WSConn adapts a *websocket.Conn to net.Conn.
//
// Read deadline semantics differ from TCP on purpose: gorilla/websocket
// treats every read error (including timeouts) as permanent and panics on
// repeated reads of a failed connection, while the reprox code relies on
// TCP-style transient timeouts (e.g. the tunnel heartbeat loop re-arms a
// 1-minute deadline and ignores timeouts). To keep both worlds working,
// SetReadDeadline is accepted but not enforced — Read blocks until a message
// arrives or the peer goes away. Fatal read errors are latched: the first
// Read returns the real error, subsequent Reads return io.EOF so loops
// terminate cleanly instead of panicking.
type WSConn struct {
	conn    *websocket.Conn
	reader  io.Reader
	readMu  sync.Mutex
	writeMu sync.Mutex
	readErr error // latched fatal read error; later reads return io.EOF
}

// NewConn wraps ws as a net.Conn. The caller owns the lifecycle;
// Close closes the underlying websocket connection.
func NewConn(ws *websocket.Conn) *WSConn {
	return &WSConn{conn: ws}
}

func (c *WSConn) Read(b []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	// A previous fatal error: terminate read loops cleanly instead of
	// hitting gorilla's "repeated read on failed connection" panic.
	if c.readErr != nil {
		return 0, io.EOF
	}

	for {
		if c.reader != nil {
			n, err := c.reader.Read(b)
			if err == io.EOF {
				c.reader = nil
				continue
			}
			if err != nil {
				c.readErr = err
				return n, err
			}
			return n, nil
		}

		op, r, err := c.conn.NextReader()
		if err != nil {
			c.readErr = err
			return 0, err
		}
		// NextReader only returns data messages; control frames
		// (ping/pong) are handled internally by the library.
		if op != websocket.BinaryMessage && op != websocket.TextMessage {
			continue
		}
		c.reader = r
	}
}

func (c *WSConn) Write(b []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := c.conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *WSConn) Close() error {
	return c.conn.Close()
}

func (c *WSConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *WSConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *WSConn) SetDeadline(t time.Time) error {
	// Read deadline intentionally not enforced (see SetReadDeadline).
	if err := c.SetWriteDeadline(t); err != nil {
		return err
	}
	return nil
}

func (c *WSConn) SetReadDeadline(t time.Time) error {
	// Intentionally not enforced: gorilla/websocket treats a read timeout
	// as a permanent failure (repeated reads panic), while callers expect
	// TCP-style transient timeouts. See the WSConn doc comment.
	// Returning nil keeps deadline-setting callers working unchanged.
	_ = t
	return nil
}

func (c *WSConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// IsWSConn reports whether conn is a *WSConn. Used by the server to apply
// transport-specific rules (e.g. rejecting tcp tunnels over websocket).
func IsWSConn(conn net.Conn) bool {
	_, ok := conn.(*WSConn)
	return ok
}
