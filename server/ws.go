package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jhonoryza/reprox/package/ws"
)

// WebSocket transport endpoints, served on the existing HTTP(S) listener:
//
//	GET /_reprox/event  -> replaces the raw TCP event channel (default port 4321)
//	GET /_reprox/data   -> replaces the raw TCP private data channel
//
// The gob event protocol is unchanged; it just runs on top of websocket
// binary messages via ws.WSConn. Enable it on the client by setting
// DOMAIN_EVENT to a ws:// or wss:// URL, e.g.
// DOMAIN_EVENT=ws://reprox.example.com:4000/_reprox/event

var wsUpgrader = websocket.Upgrader{
	// reprox has no auth on the raw TCP event channel either,
	// keep the same posture for the websocket one.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// readHTTPHead reads from conn until the end of the HTTP header block
// (\r\n\r\n), bounded to 8KB. It never over-reads: it stops exactly at the
// header end, so the rest of the stream stays intact for the next reader.
func readHTTPHead(conn net.Conn) ([]byte, error) {
	var head []byte
	one := make([]byte, 1)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := conn.Read(one)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			continue
		}
		head = append(head, one[0])
		if len(head) >= 4 && string(head[len(head)-4:]) == "\r\n\r\n" {
			break
		}
		if len(head) > 8192 {
			return nil, fmt.Errorf("http header too large")
		}
	}
	return head, nil
}

// hijackWriter is a minimal http.ResponseWriter over a raw net.Conn that
// supports hijacking, so gorilla/websocket can take over the connection.
type hijackWriter struct {
	conn net.Conn
	br   *bufio.Reader
	hdr  http.Header
}

func (w *hijackWriter) Header() http.Header {
	if w.hdr == nil {
		w.hdr = make(http.Header)
	}
	return w.hdr
}

func (w *hijackWriter) Write(p []byte) (int, error) {
	return w.conn.Write(p)
}

func (w *hijackWriter) WriteHeader(status int) {
	fmt.Fprintf(w.conn, "HTTP/1.1 %d %s\r\n", status, http.StatusText(status))
	_ = w.hdr.Write(w.conn)
	_, _ = io.WriteString(w.conn, "\r\n")
}

func (w *hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// br may still hold unread head bytes; chain it in front of conn.
	r := bufio.NewReader(io.MultiReader(w.br, w.conn))
	return w.conn, bufio.NewReadWriter(r, bufio.NewWriter(w.conn)), nil
}

// serveReproxWS upgrades the connection to websocket and dispatches it to
// the event handler or to the matching tunnel's data handler.
func (r *Reprox) serveReproxWS(conn net.Conn, head []byte) error {
	br := bufio.NewReader(bytes.NewReader(head))
	req, err := http.ReadRequest(br)
	if err != nil {
		writeResponse(conn, 400, "Bad Request", "Bad Request")
		return nil
	}

	hw := &hijackWriter{conn: conn, br: br}
	wsConn, err := wsUpgrader.Upgrade(hw, req, nil)
	if err != nil {
		// upgrader already wrote an HTTP error response
		_ = conn.Close()
		return err
	}
	wsc := ws.NewConn(wsConn)

	switch req.URL.Path {
	case ws.EventPath:
		return r.serveEvent(wsc)
	case ws.DataPath:
		return r.serveWSData(wsc, req)
	default:
		_ = wsc.Close()
		return fmt.Errorf("unknown reprox ws path %s", req.URL.Path)
	}
}

// serveWSData pairs an incoming websocket data connection with the waiting
// public connection of the tunnel identified by the ?host= query param.
// It mirrors HTTPTunnel.privateHandler's TCP flow.
func (r *Reprox) serveWSData(conn net.Conn, req *http.Request) error {
	defer conn.Close()

	host := req.URL.Query().Get("host")
	if host == "" {
		return fmt.Errorf("missing host query param")
	}
	if tunnelHost, ok := r.cnameMap[host]; ok && tunnelHost != "" {
		host = tunnelHost
	}
	host = strings.ToLower(host)
	tunnel, found := r.httpMap[host]
	if !found {
		return fmt.Errorf("unknown tunnel host %s", host)
	}
	return tunnel.privateHandler(conn)
}
