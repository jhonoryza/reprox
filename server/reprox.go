package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/jhonoryza/reprox/server/config"
	"github.com/jhonoryza/reprox/server/events"
	"github.com/jhonoryza/reprox/server/tcp"
	"github.com/jhonoryza/reprox/server/tunnel"
)

const dateFormat = "2006/01/02 15:04:05"

type Reprox struct {
	config      config.Config
	eventServer tcp.TCPServer // handle request from client app
	httpServer  tcp.TCPServer // handle request from http
	httpsServer tcp.TCPServer // handle request from https
	cnameMap    map[string]string
	httpTunnels map[string]*tunnel.HTTPTunnel
}

func (r *Reprox) Init(conf config.Config) error {
	r.config = conf
	r.cnameMap = make(map[string]string)
	r.httpTunnels = make(map[string]*tunnel.HTTPTunnel)

	// Membuka listener event server
	err := r.eventServer.Init(conf.EventServerPort, "reprox_event_server")
	if err != nil {
		return err
	}

	// Membuka listener http server
	err = r.httpServer.Init(conf.HttpServerPort, "reprox_http_server")
	if err != nil {
		return err
	}

	if conf.EnableTLS {
		// Membuka listener https server
		err = r.httpsServer.InitTLS(
			conf.HttpServerPort,
			"reprox_http_server",
			conf.TLSCertFile,
			conf.TLSKeyFile,
		)
		if err != nil {
			return err
		}
	}

	log.Println("reprox init ok")

	return nil
}

func (r *Reprox) Start() {
	go r.eventServer.Start(r.serveEventConn)
	go r.httpServer.Start(r.serveHttpConn)

	if r.config.EnableTLS {
		go r.httpsServer.Start(r.serveHttpConn)
	}

	log.Println("reprox start ok")
}

func (r *Reprox) serveEventConn(conn net.Conn) error {
	defer conn.Close()

	var event events.Event[events.TunnelRequested]
	err := event.Read(conn)
	if err != nil {
		return err
	}

	request := event.Data
	if request.Protocol != events.HTTP && request.Protocol != events.TCP {
		return events.WriteError(conn, "invalid protocol %s", request.Protocol)
	}
	if request.Subdomain == "" {
		request.Subdomain, err = generateRandomString(5)
		if err != nil {
			return err
		}
	}
	if err := validate(request.Subdomain); err != nil {
		return events.WriteError(conn, "invalid subdomain %s: %s", request.Subdomain, err.Error())
	}
	hostname := fmt.Sprintf("%s.%s", request.Subdomain, r.config.DomainName)
	if _, ok := r.httpTunnels[hostname]; ok {
		return events.WriteError(conn, "subdomain is busy: %s, try another one", request.Subdomain)
	}
	cname := request.CanonName
	if _, ok := r.cnameMap[cname]; ok && cname != "" {
		return events.WriteError(conn, "cname is busy: %s, try another one", request.CanonName)
	}

	var t tunnel.Tunnel
	var maxConsLimit = r.config.MaxConsPerTunnel

	switch request.Protocol {
	case events.HTTP:
		tn, err := tunnel.NewHTTP(hostname, conn, maxConsLimit)
		if err != nil {
			return events.WriteError(conn, "failed to create http tunnel", err.Error())
		}
		r.cnameMap[cname] = hostname
		r.httpTunnels[hostname] = tn
		defer delete(r.cnameMap, cname)
		defer delete(r.httpTunnels, hostname)
		t = tn
	}

	t.Open()
	defer t.Close()
	opened := events.Event[events.TunnelOpened]{
		Data: &events.TunnelOpened{
			Hostname:      t.Hostname(),
			Protocol:      t.Protocol(),
			PublicServer:  t.PublicServerPort(),
			PrivateServer: t.PrivateServerPort(),
		},
	}

	err = opened.Write(conn)
	if err != nil {
		return err
	}

	tunnelId := fmt.Sprintf("%s:%d", t.Hostname(), t.PublicServerPort())
	fmt.Printf("%s [tunnel-opened] %s\n", time.Now().Format(dateFormat), tunnelId)

	buffer := make([]byte, 8)

	// wait until connection is closed
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Minute))
		_, err := conn.Read(buffer)
		if err == io.EOF {
			break
		}
	}
	fmt.Printf("%s [tunnel-closed] %s\n", time.Now().Format(dateFormat), tunnelId)
	return nil
}

func (r *Reprox) serveHttpConn(conn net.Conn) error {
	_ = conn.SetReadDeadline(time.Now().Add(time.Second * 3))
	host, buffer, err := parseHost(conn)
	if err != nil || host == "" {
		writeResponse(conn, 400, "Bad Request", "Bad Request")
		return nil
	}
	tunnelHost, ok := r.cnameMap[host]
	if ok && tunnelHost != "" {
		host = tunnelHost
	}
	host = strings.ToLower(host)
	tunnel, found := r.httpTunnels[host]
	if !found {
		writeResponse(conn, 400, "Not Found", "tunnel not found")
	}
	return tunnel.HttpConnectionHandler(conn, buffer)
}

func (r *Reprox) Stop() error {
	err := r.eventServer.Stop()
	if err != nil {
		return err
	}

	err = r.httpServer.Stop()
	if err != nil {
		return err
	}

	if r.config.EnableTLS {
		err = r.httpsServer.Stop()
		if err != nil {
			return err
		}
	}

	log.Println("reprox stop ok")

	return nil
}
