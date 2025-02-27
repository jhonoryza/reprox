package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/jhonoryza/reprox/package/events"
	"github.com/jhonoryza/reprox/package/utils"
)

type Client struct {
	config    Config
	protocol  string // http or tcp
	subdomain string
	cname     string // example.com

	localServer  string // localhost:localport
	remoteServer string // private 	-> remote domain:remote port
	publicServer string // public 	-> remote domain:remote port
}

func (c *Client) Start(port uint16, targetPort uint16) {
	eventConn, err := net.Dial("tcp", c.config.Events)
	if err != nil {
		log.Fatalf("failed to connect to event server: %s\n", err)
	}
	defer eventConn.Close()

	request := events.Event[events.TunnelRequested]{
		Data: &events.TunnelRequested{
			Protocol:   c.protocol,
			Subdomain:  c.subdomain,
			CanonName:  c.cname,
			TargetPort: targetPort,
		},
	}

	err = request.Write(eventConn)
	if err != nil {
		log.Fatalf("failed to send request: %s\n", err)
	}

	var t events.Event[events.TunnelOpened]
	err = t.Read(eventConn)
	if err != nil {
		log.Fatalf("failed to receive tunnel info: %s\n", err)
	}
	if t.Data.ErrorMessage != "" {
		log.Fatalf(t.Data.ErrorMessage)
	}

	c.localServer = fmt.Sprintf("localhost:%d", port)
	c.remoteServer = fmt.Sprintf("%s:%d", c.config.Domain, t.Data.PrivateServer)
	c.publicServer = fmt.Sprintf("%s:%d", t.Data.Hostname, t.Data.PublicServer)

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

	remoteConn, err := net.Dial("tcp", c.remoteServer)
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
