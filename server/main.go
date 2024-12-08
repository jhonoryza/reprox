package main

import (
	"log"
	"os"
	"os/signal"
)

func main() {
	var conf Config
	var reprox Reprox

	err := conf.Load()
	if err != nil {
		log.Fatalf("failed to load conf: %v", err)
	}

	err = reprox.Load(conf)
	if err != nil {
		log.Fatalf("failed to load reprox %v", err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	reprox.Start()
	defer reprox.Stop()

	<-signalChan
}
