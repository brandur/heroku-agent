package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("serving ...")
	fmt.Fprintf(w, "hello from heroku-agent: %s", r.URL.Path[1:])
}

func handleSignals(l net.Listener) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		// Wait for a SIGINT or SIGKILL:
		sig := <-c
		fmt.Printf("Caught signal %s: shutting down.\n", sig)
		// Stop listening (and unlink the socket if unix type):
		l.Close()
		os.Exit(0)
	}(sigc)
}

func main() {
	http.HandleFunc("/", handler)

	l, err := net.Listen("unix", "/tmp/heroku-agent.sock")
	if err != nil {
		panic(err)
	}

	// handle common process-killing signals so we can gracefully shut down
	handleSignals(l)

	server := &http.Server{}
	err = server.Serve(l)
	if err != nil {
		panic(err)
	}
}
