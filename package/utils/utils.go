package utils

import (
	"io"
	"net"
	"time"
)

func Bind(src net.Conn, dst net.Conn, debug io.Writer) error {
	defer src.Close()
	defer dst.Close()

	buf := make([]byte, 4096)

	for {
		_ = src.SetReadDeadline(time.Now().Add(time.Second))
		n, err := src.Read(buf)
		if err == io.EOF {
			break
		}

		_ = dst.SetWriteDeadline(time.Now().Add(time.Second))
		_, err = dst.Write(buf[:n])
		if err != nil {
			return err
		}
		if debug != nil {
			debug.Write(buf[:n])
		}
		time.Sleep(time.Millisecond * 10)
	}

	return nil
}
