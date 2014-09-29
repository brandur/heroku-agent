package main

import (
	"fmt"
	"net/rpc"
	"os"
	"time"
)

func RunCommand(command string) {
	switch {
	case command == "clear":
		clear()
	case command == "help":
		help()
	case command == "state":
		stats()
	case command == "stop":
		stop()
	case command == "version":
		version()
	default:
		printUsage()
		os.Exit(1)
	}
}

func call(method string, args interface{}, reply interface{}) {
	client := getClient()

	start := time.Now()
	logStart(method)
	defer logFinish(method, start)

	err := client.Call("RpcReceiver."+method, args, reply)
	if err != nil {
		fail(err)
	}
}

func clear() {
	call("Clear", []string{}, &[]string{})
	fmt.Printf("Cleared all stores\n")
}

func getClient() *rpc.Client {
	controlPath := getControlSocketPath()
	client, err := rpc.DialHTTP("unix", controlPath)
	if err != nil {
		fail(fmt.Errorf("couldn't connect to daemon: " + err.Error()))
	}

	logger.Printf("Connecting to: %s\n", controlPath)

	return client
}

func help() {
	printUsage()
	fmt.Printf(`
Runs as daemon unless [command] is specified.

Commands:

    clear        Clear daemon's cache and two factor store
    help         Display help text
    state        Display daemon's state
    stop         Stop daemon
    version      Display version
`)
}

func logFinish(method string, start time.Time) {
	logger.Printf("[command] Response: RPC: %s [finish] [elapsed=%v]\n", method,
		time.Now().Sub(start))
}

func logStart(method string) {
	logger.Printf("[command] Request: RPC: %s [start]\n", method)
}

func stats() {
	state := &State{}
	call("GetState", []string{}, state)
	fmt.Printf("Cache count: %v\n", state.CacheCount)
	fmt.Printf("Second factor count: %v\n", state.TwoFactorCount)
	fmt.Printf("Up: %v\n", time.Now().Sub(state.UpAt))
}

func stop() {
	call("Stop", []string{}, &[]string{})
	fmt.Printf("Stopped\n")
}

func version() {
	fmt.Printf("%s\n", Version)
}
