package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
)

type TCPServer struct {
	title    string // label
	listener net.Listener
}

func (t *TCPServer) Listen(port uint16, title string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	t.title = title
	t.listener = ln
	return nil
}

func (t *TCPServer) ListenTLS(port uint16, title, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	config := tls.Config{Certificates: []tls.Certificate{cert}}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), &config)
	if err != nil {
		return err
	}
	t.title = title
	t.listener = ln
	return nil
}

func (t *TCPServer) ListenerAccept(handler func(conn net.Conn) error) {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return
		}

		go func() {
			err := handler(conn)
			if err != nil {
				log.Printf("[%s]: %s\n", t.title, err.Error())
			}
		}()
	}
}

func (t *TCPServer) ListenerStop() error {
	return t.listener.Close()
}

func (t *TCPServer) Port() uint16 {
	return uint16(t.listener.Addr().(*net.TCPAddr).Port)
}
