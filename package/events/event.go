package events

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
)

const (
	TCP  string = "tcp"
	HTTP string = "http"
)

type EventType interface {
	TunnelRequested | TunnelOpened | ConnectionReceived
}

type TunnelRequested struct {
	Protocol  string
	Subdomain string
	CanonName string
}

type TunnelOpened struct {
	Hostname      string
	Protocol      string
	PublicServer  uint16
	PrivateServer uint16
	ErrorMessage  string
}

type ConnectionReceived struct {
	ClientIP    net.IP
	ClientPort  uint16
	RateLimited bool
}

type Event[Type EventType] struct {
	Data *Type
}

func (e *Event[EventType]) encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(e.Data)
	if err != nil {
		return nil, err
	}
	data := buf.Bytes()
	return data, nil
}

func (e *Event[EventType]) decode(data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&e.Data)
	return err
}

func (e *Event[EventType]) Read(conn io.Reader) error {
	buffer := make([]byte, 2)
	_, err := conn.Read(buffer)
	if err != nil {
		return err
	}

	length := binary.LittleEndian.Uint16(buffer)
	buffer = make([]byte, length)
	_, err = conn.Read(buffer)
	if err != nil {
		return err
	}

	err = e.decode(buffer)
	return err
}

func (e *Event[EventType]) Write(conn io.Writer) error {
	data, err := e.encode()
	if err != nil {
		return err
	}

	length := make([]byte, 2)
	binary.LittleEndian.PutUint16(length, uint16(len(data)))
	_, err = conn.Write(length)
	if err != nil {
		return err
	}

	_, err = conn.Write(data)
	return err
}

func WriteError(conn io.Writer, message string, args ...string) error {
	event := Event[TunnelOpened]{
		Data: &TunnelOpened{
			ErrorMessage: fmt.Sprintf(message, args),
		},
	}

	event.Write(conn)
	return errors.New(event.Data.ErrorMessage)
}
