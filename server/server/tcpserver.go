package server

import (
	"fmt"
	"log"
	"net"
)

type TCPServer struct {
	title    string
	listener net.Listener
}

/*
 * menginisialisasi server TCP
 * Membuka listener TCP pada port yang ditentukan
 * Jika berhasil, simpan title dan listener (ln)
 * ke dalam field struct TCPServer
 */
func (t *TCPServer) Init(port uint16, title string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	t.title = title
	t.listener = ln
	return nil
}

/*
 * memulai server TCP dan menangani koneksi masuk
 * handler: Fungsi callback yang dipanggil untuk setiap koneksi masuk.
 * Fungsi ini menerima objek net.Conn (representasi koneksi)
 */
func (t *TCPServer) Start(handler func(conn net.Conn) error) {

	// Loop infinite untuk terus menerima koneksi baru
	for {

		// menerima koneksi baru yang masuk
		conn, err := t.listener.Accept()
		if err != nil {
			return
		}

		/*
		 * Untuk setiap koneksi yang diterima, sebuah goroutine baru dibuat
		 * Penggunaan goroutine memungkinkan server
		 * menangani banyak koneksi secara bersamaan
		 */
		go func() {

			// Fungsi ini menjalankan handleryang ditentukan oleh pengguna
			err := handler(conn)
			if err != nil {
				log.Printf("[%s]: %s\n", t.title, err.Error())
			}
		}()
	}
}

func (t *TCPServer) Stop() error {
	return t.listener.Close()
}

func (t *TCPServer) Port() uint16 {
	return uint16(t.listener.Addr().(*net.TCPAddr).Port)
}
