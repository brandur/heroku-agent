package main

import (
	"net/http"
	"net/http/httptest"
	"time"
)

func LogHandler(r *http.Request, next NextHandlerFunc) (*httptest.ResponseRecorder, error) {
	start := time.Now()
	logger.Printf("[log] Request: %s %s%s [start]\n", r.Method, r.Host, r.URL.String())

	// in case of an error -- keep going
	w, err := next(r)

	requestId := w.Header().Get("Request-Id")
	if requestId != "" {
		requestId = " [request_id=" + requestId + "]"
	}

	logger.Printf("[log] Request: %s %s%s [finish] [elapsed=%v] [status=%v]%s\n",
		r.Method, r.Host, r.URL.String(), time.Now().Sub(start), w.Code, requestId)

	return w, err
}
