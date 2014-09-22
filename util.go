package main

import (
	"net/http"
	"net/url"
	"strings"
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
