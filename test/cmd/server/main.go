package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	rrpctest "github.com/netgarden/rrpc/test"
)

func main() {

	var err error

	server := rrpctest.NewServer()
	err = server.Init()
	if err != nil {
		log.Fatal(err)
	}

	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	server.Stop()
}
