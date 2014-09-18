package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func LogHandler(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder {
	fmt.Printf("Request: %s %s [start]\n", r.Method, r.URL.String())

	w := next(r)

	requestId := w.Header().Get("Request-Id")
	if requestId != "" {
		requestId = " [request_id=" + requestId + "]"
	}

	fmt.Printf("Request: %s %s [finish] [status=%v]%s\n",
		r.Method, r.URL.String(), w.Code, requestId)

	return w
}
