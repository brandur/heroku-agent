package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"
)

func HandleSignals(listeners ...net.Listener) {
	sigc := make(chan os.Signal, 1)
	// wait for SIGINT, SIGKILL, or SIGTERM
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)

	sig := <-sigc
	logger.Printf("Caught signal %s: shutting down\n", sig)

	// stop listening (and unlink the socket if unix type)
	for _, listener := range listeners {
		listener.Close()
	}
	os.Exit(0)
}
