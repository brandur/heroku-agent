package main

import (
	"encoding/base64"
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

func fail(status int, err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	os.Exit(status)
}

func getPath(key string, value string) string {
	path := os.Getenv(key)
	if path == "" {
		path = value
	}

	path, err := homedir.Expand(path)
	if err != nil {
		fail(1, err)
	}

	return path
}

func getControlSocketPath() string {
	return getPath("HEROKU_AGENT_CONTROL_SOCK", DefaultControlSocketPath)
}

func getProxySocketPath() string {
	return getPath("HEROKU_AGENT_SOCK", DefaultProxySocketPath)
}

// Normalizes the `Authorization` header.
//
// This isn't strictly necessary, but the API has a number of authentication
// techniques that will work against it, and trying to make them as uniform as
// possible helps to consolidate requests across clients that might be talking
// to heroku-agent.
func normalizeAuth(rawAuth string) string {
	// First of all, check for:
	//
	//     Authorization: Bearer <token>
	//
	// If we find it, then the normalized auth is the token itself.
	token := strings.TrimPrefix(rawAuth, "Bearer ")
	if token != rawAuth {
		return token
	}

	// See if we have a "Basic" authorization:
	//
	//     Authorization: Basic <base64 encoded creds>
	//
	// If we don't, then we don't know how to normalize this authorization, so
	// just return the opaque value.
	encodedAuth := strings.TrimPrefix(rawAuth, "Basic ")
	if encodedAuth == rawAuth {
		return rawAuth
	}

	decodedAuth, err := base64.StdEncoding.DecodeString(encodedAuth)
	if err != nil {
		return rawAuth
	}

	// See if we have a basic authorization with an empty user and a token:
	//
	//     Authorization: Basic <base64 encoded ":<token>">
	//
	// If we don't, then we probably have an "<email>:<token>" or
	// "<email>:<password>", which we shouldn't provide any special handling
	// for, so return the opaque value. We would theoretically like to handle
	// the former case, but unfortunately there's no way to differentiate
	// between the two.
	creds := strings.Split(string(decodedAuth), ":")
	if len(creds) != 2 || creds[0] != "" {
		return rawAuth
	}

	return creds[1]
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
