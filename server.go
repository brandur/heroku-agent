package main

import (
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	state *State
)

type State struct {
	UpAt time.Time
}

func Serve() {
	state = &State{
		UpAt: time.Now(),
	}

	proxyPath := getPath("HEROKU_AGENT_SOCK", "~/.heroku-agent.sock")
	proxyListener := initListener(proxyPath)

	controlPath := getPath("HEROKU_AGENT_CONTROL_SOCK",
		"~/.heroku-agent-control.sock")
	controlListener := initListener(controlPath)

	rpc.Register(new(Receiver))
	rpc.HandleHTTP()
	go http.Serve(controlListener, nil)

	// handle common process-killing signals so we can gracefully shut down
	go handleSignals(proxyListener, controlListener)

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
	err := server.Serve(proxyListener)
	if err != nil {
		fail(err)
	}
}

func handleSignals(listeners ...net.Listener) {
	sigc := make(chan os.Signal, 1)
	// wait for SIGINT, SIGKILL, or SIGTERM
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)

	sig := <-sigc
	logger.Printf("Caught signal %s: shutting down\n", sig)

	// stop listening (and unlink the socket if unix type)
	for _, listener := range listeners {
		listener.Close()
	}
	os.Exit(0)
}

func initListener(socketPath string) net.Listener {
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		e, ok := err.(*net.OpError).Err.(*os.SyscallError)
		if ok && e.Err == syscall.EADDRINUSE {
			logger.Printf("heroku-agent already running at %s\n", socketPath)
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

	logger.Printf("Listening on: %s\n", socketPath)
	return l
}

type Receiver struct {
}

func (r *Receiver) Clear(_ []string, _ *[]string) error {
	start := time.Now()
	r.logStart("Clear")
	defer r.logFinish("Clear", start)

	ClearCache()
	ClearTwoFactorStore()
	return nil
}

func (r *Receiver) State(_ []string, s *State) error {
	start := time.Now()
	r.logStart("State")
	defer r.logFinish("State", start)

	s.UpAt = state.UpAt
	return nil
}

func (r *Receiver) logFinish(name string, start time.Time) {
	logger.Printf("[server] Response: RPC: %s [finish] [elapsed=%v]\n", name,
		time.Now().Sub(start))
}

func (r *Receiver) logStart(name string) {
	logger.Printf("[server] Request: RPC: %s [start]\n", name)
}
