package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Println("no command specified")
		printHelp()
	}

	switch os.Args[1] {
	case "help", "--help", "-h", "-help":
		printHelp()
	}

	protocol := ""
	command := os.Args[1]
	flags := parseFlags(os.Args[2:])

	switch command {
	case "serve":
		if flags.dir == "" {
			log.Fatalf("required --dir path")
		}
		proto, port := handleServe(flags.dir)
		flags.port = uint16(port)
		protocol = proto
	case "http", "tcp":
		protocol = command
		if flags.port <= 0 {
			log.Fatalf("invalid port or port not specified")
		}
	default:
		log.Fatalf("unknown command: %s, client --help", command)
	}

	// load client config
	var conf Config
	err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	// start client request
	client := Client{
		config:    conf,
		protocol:  protocol,
		subdomain: flags.subdomain,
		cname:     flags.cname,
	}

	go client.Start(flags.port)

	// terminate if interrupt ctrl+c detected
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}

func printHelp() {
	fmt.Printf("Usage: client <command> [arguments]\n\n")
	fmt.Println("Commands:")
	fmt.Println("  http -p <port>                Start an HTTP tunnel with a random subdomain")
	fmt.Println("  http -p <port> -s <subdomain> Start an HTTP tunnel with a custom subdomain")
	fmt.Println("  tcp -p <port> -s <subdomain> Start an HTTP tunnel with a custom subdomain")
	fmt.Println("  serve -dir <dir>                 Serve files with built-in Http Server")
	fmt.Println("  --help                      Show this help message")
	os.Exit(0)
}

type Flags struct {
	cname     string
	subdomain string
	port      uint16
	dir       string
}

func parseFlags(args []string) Flags {
	var flags Flags
	for i, arg := range args {
		switch arg {
		case "-s", "-subdomain", "--subdomain":
			flags.subdomain = args[i+1]
		case "-c", "-cname", "--cname":
			flags.cname = args[i+1]
		case "-p", "--port", "-port":
			port, err := strconv.Atoi(args[i+1])
			if err != nil {
				log.Fatalf("invalid port %v", err)
			}
			flags.port = uint16(port)
		case "-dir", "--dir":
			flags.dir = args[i+1]
		}
	}
	return flags
}

func handleServe(dir string) (string, int) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("no such dir %s", err)
	}

	handler := http.FileServer(http.Dir(dir))

	// listen on random port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to start file server: %s", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		err = http.Serve(listener, handler)
		if err != nil {
			log.Fatalf("cannot serve files on %s:%s", dir, err)
		}
	}()

	time.AfterFunc(time.Millisecond*600, func() {
		log.Println("serving: \t", dir)
	})

	return "http", port
}
