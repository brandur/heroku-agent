package main

import (
	"net"
	"net/http"
	"net/rpc"
	"os"
	"syscall"
	"time"
)

var (
	state *State
)

type Receiver struct {
}

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
	go HandleSignals(proxyListener, controlListener)

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

func (r *Receiver) Clear(_ []string, _ *[]string) error {
	start := time.Now()
	logger.Printf("[server] Request: RPC: Clear [start]\n")
	defer func() {
		logger.Printf("[server] Response: RPC: Clear [finish] [elapsed=%v]\n",
			time.Now().Sub(start))
	}()

	ClearCache()
	ClearTwoFactorStore()
	return nil
}

func (r *Receiver) State(_ []string, s *State) error {
	start := time.Now()
	logger.Printf("[server] Request: RPC: State [start]\n")
	defer func() {
		logger.Printf("[server] Response: RPC: State [finish] [elapsed=%v]\n",
			time.Now().Sub(start))
	}()

	s.UpAt = state.UpAt
	return nil
}
