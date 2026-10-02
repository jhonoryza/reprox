package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jhonoryza/reprox/package/events"
	"github.com/jhonoryza/reprox/package/utils"
	"github.com/jhonoryza/reprox/package/ws"
)

type Client struct {
	config    Config
	protocol  string // http or tcp
	subdomain string
	cname     string // example.com

	localServer  string // localhost:localport
	remoteServer string // private 	-> remote domain:remote port
	publicServer string // public 	-> remote domain:remote port

	// websocket transport state; active when DOMAIN_EVENT is a ws(s):// URL
	wsMode     bool
	wsDataBase string // e.g. ws://domain:4000/_reprox/data
	hostname   string // assigned tunnel hostname from the server
}

// isWebsocketURL reports whether the event endpoint is a websocket URL.
func isWebsocketURL(endpoint string) bool {
	return strings.HasPrefix(endpoint, "ws://") || strings.HasPrefix(endpoint, "wss://")
}

// wsDialer dials through the environment proxy when set, which is required
// on networks where raw TCP egress is blocked.
//
// For plain ws:// URLs we tunnel with an explicit CONNECT request: Go's
// http.Transport would otherwise send the handshake to the proxy in
// forward-proxy mode (GET ws://...), which egress proxies don't understand
// and just hang on. wss:// already goes through CONNECT by default, so it
// keeps the standard proxy handling.
func wsDialer(rawURL string) *websocket.Dialer {
	d := &websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}
	if u, err := url.Parse(rawURL); err == nil && u.Scheme == "ws" {
		host, _, _ := net.SplitHostPort(u.Host)
		isLoopback := host == "localhost" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
		switch {
		case isLoopback:
			return d // direct dial, no proxy
		case proxyFromEnv() != "":
			d.NetDialContext = proxyConnectDial
			return d
		}
	}
	d.Proxy = http.ProxyFromEnvironment
	return d
}

// proxyFromEnv returns the configured HTTP proxy URL, or "" if none.
func proxyFromEnv() string {
	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// proxyConnectDial dials addr through the configured HTTP proxy with an
// explicit CONNECT request and returns the raw tunnelled connection, over
// which the caller performs the websocket handshake.
func proxyConnectDial(ctx context.Context, network, addr string) (net.Conn, error) {
	proxyURL, err := url.Parse(proxyFromEnv())
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}
	conn, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", proxyURL.Host)
	if err != nil {
		return nil, err
	}
	// Bound the CONNECT handshake; without this a silent proxy hangs forever.
	conn.SetDeadline(time.Now().Add(20 * time.Second))
	defer conn.SetDeadline(time.Time{})

	var sb strings.Builder
	fmt.Fprintf(&sb, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
	if proxyURL.User != nil {
		pw, _ := proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(proxyURL.User.Username() + ":" + pw))
		fmt.Fprintf(&sb, "Proxy-Authorization: Basic %s\r\n", auth)
	}
	sb.WriteString("\r\n")

	if _, err := conn.Write([]byte(sb.String())); err != nil {
		conn.Close()
		return nil, err
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("proxy CONNECT failed: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("proxy CONNECT rejected: %s", resp.Status)
	}
	if br.Buffered() > 0 {
		conn = &bufferedConn{Conn: conn, br: br}
	}
	return conn, nil
}

// bufferedConn drains bytes already buffered during CONNECT before reading
// from the underlying connection.
type bufferedConn struct {
	net.Conn
	br *bufio.Reader
}

func (c *bufferedConn) Read(b []byte) (int, error) {
	if c.br.Buffered() > 0 {
		return c.br.Read(b)
	}
	return c.Conn.Read(b)
}

func (c *Client) dialEvent() net.Conn {
	if isWebsocketURL(c.config.Events) {
		wsConn, _, err := wsDialer(c.config.Events).Dial(c.config.Events, nil)
		if err != nil {
			log.Fatalf("failed to connect to event server: %s\n", err)
		}
		c.wsMode = true
		// data endpoint shares the event URL's scheme+host
		if u, err := url.Parse(c.config.Events); err == nil {
			u.Path = ws.DataPath
			u.RawQuery = ""
			u.Fragment = ""
			c.wsDataBase = u.String()
		}
		fmt.Printf("Transport: \t WebSocket \n")
		return ws.NewConn(wsConn)
	}

	eventConn, err := net.Dial("tcp", c.config.Events)
	if err != nil {
		log.Fatalf("failed to connect to event server: %s\n", err)
	}
	return eventConn
}

func (c *Client) dialData() (net.Conn, error) {
	if c.wsMode {
		dataURL := c.wsDataBase + "?host=" + url.QueryEscape(c.hostname)
		wsConn, _, err := wsDialer(dataURL).Dial(dataURL, nil)
		if err != nil {
			return nil, err
		}
		return ws.NewConn(wsConn), nil
	}
	return net.Dial("tcp", c.remoteServer)
}

func (c *Client) Start(port uint16, targetPort uint16) {
	eventConn := c.dialEvent()
	defer eventConn.Close()

	request := events.Event[events.TunnelRequested]{
		Data: &events.TunnelRequested{
			Protocol:   c.protocol,
			Subdomain:  c.subdomain,
			CanonName:  c.cname,
			TargetPort: targetPort,
		},
	}

	err := request.Write(eventConn)
	if err != nil {
		log.Fatalf("failed to send request: %s\n", err)
	}

	var t events.Event[events.TunnelOpened]
	err = t.Read(eventConn)
	if err != nil {
		log.Fatalf("failed to receive tunnel info: %s\n", err)
	}
	if t.Data.ErrorMessage != "" {
		log.Fatal(t.Data.ErrorMessage)
	}

	c.localServer = fmt.Sprintf("localhost:%d", port)
	c.remoteServer = fmt.Sprintf("%s:%d", c.config.Domain, t.Data.PrivateServer)
	c.publicServer = fmt.Sprintf("%s:%d", t.Data.Hostname, t.Data.PublicServer)
	c.hostname = t.Data.Hostname

	if c.protocol == "http" {
		c.publicServer = fmt.Sprintf("https://%s", t.Data.Hostname)
	}

	fmt.Printf("Status: \t Online \n")
	fmt.Printf("Protocol: \t %s \n", strings.ToUpper(c.protocol))
	fmt.Printf("Forwarded: \t %s -> %s \n", strings.TrimSuffix(c.publicServer, ":80"), c.localServer)

	var event events.Event[events.ConnectionReceived]
	for {
		err = event.Read(eventConn)
		if err != nil {
			log.Fatalf("failed to receive connection-received event: %s\n", err)
		}
		go c.handleEvent(*event.Data)
	}
}

func (c *Client) handleEvent(event events.ConnectionReceived) {
	localConn, err := net.Dial("tcp", c.localServer)
	if err != nil {
		log.Printf("failed to connect to local server: %s\n", err)
		return
	}
	defer localConn.Close()

	remoteConn, err := c.dialData()
	if err != nil {
		log.Printf("failed to connect to remote server: %s\n", err)
		return
	}
	defer remoteConn.Close()

	buffer := make([]byte, 2)
	binary.LittleEndian.PutUint16(buffer, event.ClientPort)
	remoteConn.Write(buffer)

	go utils.Bind(localConn, remoteConn, nil)
	utils.Bind(remoteConn, localConn, nil)
}
