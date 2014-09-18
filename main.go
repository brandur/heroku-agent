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

type HandlerFunc func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)

func buildHandlerChain(handlers []HandlerFunc) func(w http.ResponseWriter, r *http.Request) {
	next := func(_ http.ResponseWriter, _ *http.Request) {
	}

	// move through handlers in reverse and compose them on top of each other
	for i := len(handlers) - 1; i >= 0; i-- {
		handler := handlers[i]
		next = func(w http.ResponseWriter, r *http.Request) {
			handler(w, r, next)
		}
	}

	return next
}

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

	handlers := []HandlerFunc{
		CacheHandler,
	}

	http.HandleFunc("/", buildHandlerChain(handlers))

	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	// We rely on file access to guarantee security so make sure that the
	// socket is opened in the user's home directory.
	l, err := net.Listen("unix", home+"/.heroku-agent.sock")
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
