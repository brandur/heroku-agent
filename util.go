package main

import (
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	DefaultControlSocketPath = "~/.heroku-agent-control.sock"
	DefaultProxySocketPath   = "~/.heroku-agent.sock"
)

//
// Contains any functions that are called by multiple modules, but don't belong
// in any in particular.
//

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

func getPath(key string, value string) string {
	path := os.Getenv(key)
	if path == "" {
		path = value
	}

	path, err := homedir.Expand(path)
	if err != nil {
		fail(err)
	}

	return path
}

func getControlSocketPath() string {
	return getPath("HEROKU_AGENT_CONTROL_SOCK", DefaultControlSocketPath)
}

func getProxySocketPath() string {
	return getPath("HEROKU_AGENT_SOCK", DefaultProxySocketPath)
}

func printUsage() {
	fmt.Printf("Usage: heroku-agent [-v] [command]\n")
}

// Unfortunately, the Toolbelt sends a user's password via query parameter,
// which shows up in a stringified URL. This method scrubs that out for safe
// display on-screen and in-logs.
func safeUrl(u *url.URL) string {
	password := u.Query().Get("password")
	s := u.String()
	if password != "" {
		s = strings.Replace(s, password, "[scrubbed]", 1)
	}
	return s
}
