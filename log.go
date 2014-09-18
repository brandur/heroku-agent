package main

import (
	"fmt"
	"net/http"
)

func LogHandler(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	fmt.Printf("Request: %s %s [start]\n", r.Method, r.URL.String())

	next(w, r)

	// A better way to get status would be good because the presence of the
	// `Status` header isn't reliable
	status := w.Header().Get("Status")
	if status != "" {
		status = " [" + status + "]"
	}

	requestId := w.Header().Get("Request-Id")
	if requestId != "" {
		requestId = " [request_id=" + requestId + "]"
	}

	fmt.Printf("Request: %s %s [finish]%s%s\n",
		r.Method, r.URL.String(), status, requestId)
}
