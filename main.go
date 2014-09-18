package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	cache  *RequestCache
	client *http.Client
)

func copyHeaders(source http.Header, destination http.Header) {
	for h, vs := range source {
		for _, v := range vs {
			destination.Set(h, v)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	cached, isCached := cache.getCache(r)

	// don't try our cache is the client sent their own cache attempt
	if _, ok := r.Header["If-None-Match"]; ok {
		isCached = false
	}

	url := "https://" + r.Host + r.URL.String()
	req, err := http.NewRequest(r.Method, url, r.Body)

	if isCached {
		req.Header.Set("If-None-Match", cached.etag)
	}

	copyHeaders(r.Header, req.Header)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	for h, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Set(h, v)
		}
	}
	if isCached && resp.StatusCode == 304 {
		// remove headers that may be inaccurate on a cached response
		for k, _ := range contentHeaders {
			w.Header().Del(k)
		}
		copyHeaders(cached.header, w.Header())

		w.WriteHeader(200)
		w.Write(cached.content)
	} else {
		w.WriteHeader(resp.StatusCode)
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		cache.setCache(r, resp, bytes)
		w.Write(bytes)
	}

	fmt.Printf("served: %s %s (%v)\n", r.Method, r.URL.Path, resp.StatusCode)
}

func handleSignals(l net.Listener) {
	sigc := make(chan os.Signal, 1)
	// wait for a SIGINT or SIGKILL
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
