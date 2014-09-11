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

func handler(w http.ResponseWriter, r *http.Request) {
	cached, isCached := cache.getCache(r)

	// don't try our cache is the client sent their own cache attempt
	if _, ok := r.Header["If-None-Match"]; ok {
		isCached = false
	}

	req, err := http.NewRequest(r.Method, "https://api.heroku.com"+r.URL.Path+"?"+r.URL.RawQuery, r.Body)

	if isCached {
		req.Header.Set("If-None-Match", cached.etag)
	}

	for h, vs := range r.Header {
		for _, v := range vs {
			req.Header.Set(h, v)
		}
	}

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

		for h, vs := range cached.header {
			for _, v := range vs {
				w.Header().Set(h, v)
			}
		}

		w.WriteHeader(200)
		w.Write(cached.content)
	} else {
		w.WriteHeader(resp.StatusCode)
		//io.Copy(w, resp.Body)
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
	cache = newRequestCache()
	client = &http.Client{}

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
