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
	StopChan chan bool
	UpAt     time.Time
}

func Serve() {
	state = &State{
		UpAt:     time.Now(),
		StopChan: make(chan bool),
	}

	proxyListener := initListener(getProxySocketPath())
	controlListener := initListener(getControlSocketPath())

	// register and start serving on the control socket so that a heroku-agent
	// running in "command mode" can connect and make a call
	rpc.Register(&RpcReceiver{
		State: state,
	})
	rpc.HandleHTTP()
	go http.Serve(controlListener, nil)

	// allow graceful shutdown; this is important because Unix domain sockets
	// will not clean themselves up
	go handleStop(state.StopChan, proxyListener, controlListener)

	// handle common process-killing signals
	go handleSignals(state.StopChan)

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

func handleSignals(StopChan chan bool) {
	sigc := make(chan os.Signal, 1)
	// wait for SIGINT, SIGKILL, or SIGTERM
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)

	sig := <-sigc
	logger.Printf("Caught signal %s: shutting down\n", sig)
	StopChan <- true
}

func handleStop(StopChan chan bool, listeners ...net.Listener) {
	<-StopChan

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

	logger.Printf("[server] Listening on: %s\n", socketPath)
	return l
}

// this type does nothing more than act as a target for our control plane RPC
type RpcReceiver struct {
	State *State
}

func (r *RpcReceiver) Clear(_ []string, _ *[]string) error {
	start := time.Now()
	r.logStart("Clear")
	defer r.logFinish("Clear", start)

	ClearCache()
	ClearTwoFactorStore()
	return nil
}

func (r *RpcReceiver) GetState(_ []string, s *State) error {
	start := time.Now()
	r.logStart("State")
	defer r.logFinish("State", start)

	s.UpAt = state.UpAt
	return nil
}

func (r *RpcReceiver) Stop(_ []string, _ *[]string) error {
	start := time.Now()
	r.logStart("Stop")
	defer r.logFinish("Stop", start)

	logger.Printf("[rpc] Stopping by instruction of RPC command\n")
	r.State.StopChan <- true
	return nil
}

func (r *RpcReceiver) logFinish(method string, start time.Time) {
	logger.Printf("[rpc] Response: RPC: %s [finish] [elapsed=%v]\n", method,
		time.Now().Sub(start))
}

func (r *RpcReceiver) logStart(method string) {
	logger.Printf("[rpc] Request: RPC: %s [start]\n", method)
}
