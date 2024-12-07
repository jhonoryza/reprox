package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/jhonoryza/reprox/server/config"
)

func main() {
	var conf config.Config
	var reprox Reprox

	// load default configuration
	err := conf.Load()
	if err != nil {
		log.Fatalf("failed to load conf: %v", err)
	}

	// initialize reprox
	err = reprox.Init(conf)
	if err != nil {
		log.Fatalf("failed to init reprox %v", err)
	}

	/**
	 * deteksi interupt signal ctrl+c
	 * jika terjadi terminate
	 */
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// start reprox
	reprox.Start()
	defer reprox.Stop()

	<-signalChan
}
