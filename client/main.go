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
	case "help", "--help":
		printHelp()
	}

	if len(os.Args) < 3 {
		log.Println("no arg supplied")
		printHelp()
	}

	protocol, port := "", 0
	command, arg := os.Args[1], os.Args[2]
	flags := parseFlags(os.Args[3:])

	// parse protocol and port from command
	switch command {
	case "serve":
		protocol, port = handleServe(arg)
	case "http", "tcp":
		protocol = command
		port, _ = strconv.Atoi(arg)
	default:
		log.Fatalf("unknown command: %s, client --help", command)
	}

	// validate invalid port number
	if port <= 0 {
		log.Fatalf("port number must be positive int")
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

	go client.Start(uint16(port))

	// terminate if interrupt ctrl+c detected
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}

func printHelp() {
	fmt.Printf("Usage: client <command> [arguments]\n\n")
	fmt.Println("Commands:")
	fmt.Println("  http  <port>                Start an HTTP tunnel on the specified port")
	fmt.Println("  http  <port> -s <subdomain> Start an HTTP tunnel with a custom subdomain")
	fmt.Println("  serve <dir>                 Serve files with built-in Http Server")
	fmt.Println("  --help                      Show this help message")
	os.Exit(0)
}

type Flags struct {
	cname     string
	subdomain string
}

func parseFlags(args []string) Flags {
	var flags Flags
	for i, arg := range args {
		switch arg {
		case "-s", "-subdomain", "--subdomain":
			flags.subdomain = args[i+1]
		case "-c", "-cname", "--cname":
			flags.cname = args[i+1]
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
