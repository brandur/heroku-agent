package main

import (
	"fmt"
	"net/rpc"
	"time"
)

func Run(command string) {
	controlPath := getPath("HEROKU_AGENT_CONTROL_SOCK",
		"~/.heroku-agent-control.sock")
	client, err := rpc.DialHTTP("unix", controlPath)
	if err != nil {
		fail(fmt.Errorf("couldn't connect to server: " + err.Error()))
	}

	switch {
	case command == "clear":
		err = clear(client)
	case command == "state":
		err = stats(client)
	default:
		printUsage()
	}

	if err != nil {
		fail(err)
	}
}

func clear(client *rpc.Client) error {
	err := client.Call("Receiver.Clear", []string{}, &[]string{})
	if err != nil {
		return err
	}
	fmt.Printf("Cleared all stores\n")
	return nil
}

func stats(client *rpc.Client) error {
	state := &State{}
	err := client.Call("Receiver.State", []string{}, state)
	if err != nil {
		return err
	}
	fmt.Printf("Up: %v\n", time.Now().Sub(state.UpAt))
	return nil
}
