package main

import (
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
)

type HandlerFunc func(w *httptest.ResponseRecorder, r *http.Request, next NextHandlerFunc)

type NextHandlerFunc func(w *httptest.ResponseRecorder, r *http.Request)

func buildHandlerChain(handlers []HandlerFunc) func(w http.ResponseWriter, r *http.Request) {
	chain := func(_ *httptest.ResponseRecorder, _ *http.Request) {
	}

	// move through handlers in reverse and compose them on top of each other
	for i := len(handlers) - 1; i >= 0; i-- {
		handler := handlers[i]
		next := chain
		chain = func(w *httptest.ResponseRecorder, r *http.Request) {
			handler(w, r, next)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		recorder := httptest.NewRecorder()
		chain(recorder, r)
		copyHeaders(recorder.Header(), w.Header())
		w.WriteHeader(recorder.Code)
		w.Write(recorder.Body.Bytes())
	}
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
		LogHandler,
		CacheHandler,
		ProxyHandler,
	}

	http.HandleFunc("/", buildHandlerChain(handlers))

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
