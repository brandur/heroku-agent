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
	case command == "version":
		version()
	default:
		printUsage()
		os.Exit(1)
	}
}

func clear() {
	client := getClient()
	err := client.Call("Receiver.Clear", []string{}, &[]string{})
	if err != nil {
		fail(err)
	}
	fmt.Printf("Cleared all stores\n")
}

func getClient() *rpc.Client {
	controlPath := getControlSocketPath()
	client, err := rpc.DialHTTP("unix", controlPath)
	if err != nil {
		fail(fmt.Errorf("couldn't connect to server: " + err.Error()))
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
    version      Display version
`)
}

func stats() {
	client := getClient()
	state := &State{}
	err := client.Call("Receiver.State", []string{}, state)
	if err != nil {
		fail(err)
	}
	fmt.Printf("Up: %v\n", time.Now().Sub(state.UpAt))
}

func version() {
	fmt.Printf("%s\n", Version)
}
