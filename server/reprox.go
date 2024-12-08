package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/jhonoryza/reprox/package/events"
)

type Reprox struct {
	config   Config
	tcpEvent TCPServer              // handle request, from client app
	tcpHttp  TCPServer              // handle request, from http
	tcpHttps TCPServer              // handle request, from https
	cnameMap map[string]string      // mapping cname, example.com : subdomain.domain
	httpMap  map[string]*HTTPTunnel // mapping http, subdomain.domain : tunnel http
	tcpMap   map[uint16]*TCPTunnel  // mapping tcp, public port : tunnel tcp
}

func (r *Reprox) Load(conf Config) error {
	r.config = conf
	r.cnameMap = make(map[string]string)
	r.httpMap = make(map[string]*HTTPTunnel)
	r.tcpMap = make(map[uint16]*TCPTunnel)

	err := r.tcpEvent.Listen(conf.EventServerPort, "reprox_event_server")
	if err != nil {
		return err
	}

	err = r.tcpHttp.Listen(conf.HttpServerPort, "reprox_http_server")
	if err != nil {
		return err
	}

	if conf.EnableTLS {
		err = r.tcpHttps.ListenTLS(
			conf.HttpsServerPort,
			"reprox_https_server",
			conf.TLSCertFile,
			conf.TLSKeyFile,
		)
		if err != nil {
			return err
		}
	}

	log.Println("reprox load ok")

	return nil
}

func (r *Reprox) Start() {
	go r.tcpEvent.ListenerAccept(r.serveEvent)
	go r.tcpHttp.ListenerAccept(r.serveHttp)

	if r.config.EnableTLS {
		go r.tcpHttps.ListenerAccept(r.serveHttp)
	}

	log.Println("reprox start ok")
}

func (r *Reprox) Stop() error {
	err := r.tcpEvent.ListenerStop()
	if err != nil {
		return err
	}

	err = r.tcpHttp.ListenerStop()
	if err != nil {
		return err
	}

	log.Println("reprox stop ok")

	return nil
}

const dateFormat = "2006/01/02 15:04:05"

func (r *Reprox) serveEvent(conn net.Conn) error {
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
		request.Subdomain, err = generateRandomString(10)
		if err != nil {
			return err
		}
	}
	if err := validate(request.Subdomain); err != nil {
		return events.WriteError(conn, "invalid subdomain %s: %s", request.Subdomain, err.Error())
	}
	hostname := fmt.Sprintf("%s.%s", request.Subdomain, r.config.DomainName)
	if _, ok := r.httpMap[hostname]; ok {
		return events.WriteError(conn, "subdomain is busy: %s, try another one", request.Subdomain)
	}
	cname := request.CanonName
	if _, ok := r.cnameMap[cname]; ok && cname != "" {
		return events.WriteError(conn, "cname is busy: %s, try another one", request.CanonName)
	}

	if request.Protocol == events.TCP {
		return r.createNewTCPTunnel(hostname, conn)
	}

	return r.createNewHTTPTunnel(hostname, cname, conn)
}

func (r *Reprox) createNewHTTPTunnel(hostname string, cname string, conn net.Conn) error {
	tn, err := NewHTTP(hostname, conn)
	if err != nil {
		return events.WriteError(conn, "failed to create http tunnel", err.Error())
	}
	r.cnameMap[cname] = hostname
	r.httpMap[hostname] = tn
	defer delete(r.cnameMap, cname)
	defer delete(r.httpMap, hostname)

	tn.Open()
	defer tn.Close()
	opened := events.Event[events.TunnelOpened]{
		Data: &events.TunnelOpened{
			Hostname:      tn.Hostname(),
			Protocol:      tn.Protocol(),
			PublicServer:  tn.PublicServerPort(),
			PrivateServer: tn.PrivateServerPort(),
		},
	}

	err = opened.Write(conn)
	if err != nil {
		return err
	}

	tunnelId := fmt.Sprintf("%s:%d", tn.Hostname(), tn.PublicServerPort())
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

func (r *Reprox) createNewTCPTunnel(hostname string, conn net.Conn) error {
	tn, err := NewTCP(hostname, conn)
	if err != nil {
		return events.WriteError(conn, "failed to create tcp tunnel", err.Error())
	}
	r.tcpMap[tn.PublicServerPort()] = tn
	defer delete(r.tcpMap, tn.PublicServerPort())

	tn.Open()
	defer tn.Close()
	opened := events.Event[events.TunnelOpened]{
		Data: &events.TunnelOpened{
			Hostname:      tn.Hostname(),
			Protocol:      tn.Protocol(),
			PublicServer:  tn.PublicServerPort(),
			PrivateServer: tn.PrivateServerPort(),
		},
	}

	err = opened.Write(conn)
	if err != nil {
		return err
	}

	tunnelId := fmt.Sprintf("%s:%d", tn.Hostname(), tn.PublicServerPort())
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

func (r *Reprox) serveHttp(conn net.Conn) error {
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
	tunnel, found := r.httpMap[host]
	if !found {
		writeResponse(conn, 400, "Not Found", "tunnel not found")
		return fmt.Errorf("unknown host requested %s", host)
	}
	return tunnel.publicHandler(conn, buffer)
}
