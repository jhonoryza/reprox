package main

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/jhonoryza/reprox/server/config"
	"github.com/jhonoryza/reprox/server/events"
	"github.com/jhonoryza/reprox/server/server"
	"github.com/jhonoryza/reprox/server/tunnel"
)

const dateFormat = "2006/01/02 15:04:05"

type Reprox struct {
	config      config.Config
	eventServer server.TCPServer
	cnameMap    map[string]string
	httpTunnels map[string]*tunnel.HttpTunnel
}

func (r *Reprox) Init(conf config.Config) error {
	r.config = conf

	// Membuka listener TCP pada port yang ditentukan
	err := r.eventServer.Init(conf.EventServerPort, "reprox_event_server")
	if err != nil {
		return err
	}

	r.cnameMap = make(map[string]string)
	r.httpTunnels = make(map[string]*tunnel.HttpTunnel)

	return nil
}

func (r *Reprox) Start() {
	go r.eventServer.Start(r.serveEventConn)
}

func (j *Reprox) serveEventConn(conn net.Conn) error {
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
	hostname := fmt.Sprintf("%s.%s", request.Subdomain, j.config.DomainName)
	if _, ok := j.httpTunnels[hostname]; ok {
		return events.WriteError(conn, "subdomain is busy: %s, try another one", request.Subdomain)
	}
	cname := request.CanonName
	if _, ok := j.cnameMap[cname]; ok && cname != "" {
		return events.WriteError(conn, "cname is busy: %s, try another one", request.CanonName)
	}

	var t tunnel.Tunnel
	var maxConsLimit = j.config.MaxConsPerTunnel

	switch request.Protocol {
	case events.HTTP:
		tn, err := tunnel.NewHTTP(hostname, conn, maxConsLimit)
		if err != nil {
			return events.WriteError(conn, "failed to create http tunnel", err.Error())
		}
		j.cnameMap[cname] = hostname
		j.httpTunnels[hostname] = tn
		defer delete(j.cnameMap, cname)
		defer delete(j.httpTunnels, hostname)
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

func (r *Reprox) Stop() error {
	err := r.eventServer.Stop()
	if err != nil {
		return err
	}

	return nil
}
