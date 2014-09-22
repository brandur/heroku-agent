package main

import (
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	flag "github.com/ogier/pflag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
)

var (
	logger *log.Logger
)

func copyHeaders(source http.Header, destination http.Header) {
	for h, vs := range source {
		for _, v := range vs {
			destination.Set(h, v)
		}
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	os.Exit(1)
}

func init() {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: 1 * time.Minute,
		}).Dial,
	}
	client = &http.Client{
		// More aggressive timeout to minimize waits on a bad connection
		Timeout:   10 * time.Second,
		Transport: transport,
	}
}

func initLogger(verbose bool) *log.Logger {
	var writer io.Writer
	if verbose {
		writer = os.Stdout
	} else {
		writer = ioutil.Discard
	}
	return log.New(writer, "[heroku-agent] ", log.Ltime)
}

func initListener() net.Listener {
	socketPath := os.Getenv("HEROKU_AGENT_SOCK")
	if socketPath == "" {
		socketPath = "~/.heroku-agent.sock"
	}

	socketPath, err := homedir.Expand(socketPath)
	if err != nil {
		fail(err)
	}

	l, err := net.Listen("unix", socketPath)
	switch e, ok := err.(*net.OpError).Err.(*os.SyscallError); {
	case ok && e.Err == syscall.EADDRINUSE:
		logger.Printf("heroku-agent already running at %s\n", socketPath)
		os.Exit(0)
	case err != nil:
		fail(err)
	}

	// Make sure that only the current user can gain access to this socket as
	// it will hold secrets.
	err = os.Chmod(socketPath, 0600)
	if err != nil {
		fail(err)
	}

	logger.Printf("Listening on: %s\n", socketPath)
	return l
}

func main() {
	verbose := flag.BoolP("verbose", "v", false, "Verbose mode")
	flag.Parse()

	logger = initLogger(*verbose)
	listener := initListener()

	// handle common process-killing signals so we can gracefully shut down
	go HandleSignals(listener)

	// periodically reap the cache and second factor store so that we don't
	// bloat out of control
	go ReapCache()
	go ReapTwoFactorStore()

	http.HandleFunc("/", BuildHandlerChain([]HandlerFunc{
		LogHandler,
		ErrorHandler,
		TwoFactorHandler,
		CacheHandler,
		ProxyHandler,
	}))

	server := &http.Server{}
	err := server.Serve(listener)
	if err != nil {
		fail(err)
	}
}
