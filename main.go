package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func handler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}

	req, err := http.NewRequest(r.Method, "https://api.heroku.com"+r.URL.Path+"?"+r.URL.RawQuery, r.Body)
	for h, vs := range r.Header {
		for _, v := range vs {
			req.Header.Set(h, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	for h, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	fmt.Printf("served: %s (%v)\n", r.URL.Path, resp.StatusCode)
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

	//l, err := net.Listen("unix", "/tmp/heroku-agent.sock")
	l, err := net.Listen("tcp", ":2323")
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
