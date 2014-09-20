package main

import (
	"net/http"
	"net/http/httptest"
)

func LogHandler(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder {
	logger.Printf("Request: %s %s%s [start]\n", r.Method, r.Host, r.URL.String())

	w := next(r)

	requestId := w.Header().Get("Request-Id")
	if requestId != "" {
		requestId = " [request_id=" + requestId + "]"
	}

	logger.Printf("Request: %s %s%s [finish] [status=%v]%s\n",
		r.Method, r.Host, r.URL.String(), w.Code, requestId)

	return w
}
