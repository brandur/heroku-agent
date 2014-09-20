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
	"os/signal"
	"strings"
	"syscall"
)

var (
	logger *log.Logger
)

func handleSignals(l net.Listener) {
	sigc := make(chan os.Signal, 1)
	// wait for SIGINT, SIGKILL, or SIGTERM
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	sig := <-sigc
	logger.Printf("Caught signal %s: shutting down\n", sig)
	// stop listening (and unlink the socket if unix type)
	l.Close()
	os.Exit(0)
}

func fail(err error) {
	fmt.Printf("Error: %s\n", err.Error())
	os.Exit(1)
}

func init() {
	client = &http.Client{}
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
	if err != nil {
		// it would be nice to have a better way than string matching to detect
		// this error type
		if strings.Contains(err.Error(), "address already in use") {
			fmt.Printf("heroku-agent already running at %s\n", socketPath)
			os.Exit(0)
		}
		fail(err)
	}

	// Make sure that only the current user can gain access to this socket as
	// it will hold secrets.
	err = os.Chmod(socketPath, 0600)
	if err != nil {
		fail(err)
	}

	logger.Printf("Serving on: %s\n", socketPath)
	return l
}

func main() {
	verbose := flag.BoolP("verbose", "v", false, "Verbose mode")
	flag.Parse()

	logger = initLogger(*verbose)
	listener := initListener()

	// handle common process-killing signals so we can gracefully shut down
	go handleSignals(listener)

	// periodically reap the cache so that we don't bloat out of control
	go ReapCache()

	http.HandleFunc("/", BuildHandlerChain([]HandlerFunc{
		LogHandler,
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
