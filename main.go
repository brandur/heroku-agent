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

func init() {
	client = &http.Client{}
}

func main() {
	socketPath := os.Getenv("HEROKU_AGENT_SOCK")
	if socketPath == "" {
		socketPath = "~/.heroku-agent.sock"
	}

	socketPath, err := homedir.Expand(socketPath)
	if err != nil {
		panic(err)
	}

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}

	// Make sure that only the current user can gain access to this socket as
	// it will hold secrets.
	err = os.Chmod(socketPath, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Serving on: %s\n", socketPath)

	// handle common process-killing signals so we can gracefully shut down
	handleSignals(l)

	ReapCache()

	http.HandleFunc("/", BuildHandlerChain([]HandlerFunc{
		LogHandler,
		TwoFactorHandler,
		CacheHandler,
		ProxyHandler,
	}))

	server := &http.Server{}
	err = server.Serve(l)
	if err != nil {
		panic(err)
	}
}
