package main

import (
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func handleSignals(l net.Listener) {
	sigc := make(chan os.Signal, 1)
	// wait for SIGINT, SIGKILL, or SIGTERM
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		fmt.Printf("Caught signal %s: shutting down.\n", sig)
		// stop listening (and unlink the socket if unix type)
		l.Close()
		os.Exit(0)
	}(sigc)
}

func main() {
	cache = newRequestCache()
	client = &http.Client{}

	http.HandleFunc("/", BuildHandlerChain([]HandlerFunc{
		LogHandler,
		CacheHandler,
		ProxyHandler,
	}))

	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	// We rely on file access to guarantee security so make sure that the
	// socket is opened in the user's home directory.
	path := home + "/.heroku-agent.sock"
	l, err := net.Listen("unix", path)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Serving on: %s\n", path)

	// handle common process-killing signals so we can gracefully shut down
	handleSignals(l)

	server := &http.Server{}
	err = server.Serve(l)
	if err != nil {
		panic(err)
	}
}
