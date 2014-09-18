package main

import (
	"net/http"
	"net/http/httptest"
)

type HandlerFunc func(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder

type NextHandlerFunc func(r *http.Request) *httptest.ResponseRecorder

func BuildHandlerChain(handlers []HandlerFunc) func(w http.ResponseWriter, r *http.Request) {
	chain := func(_ *http.Request) *httptest.ResponseRecorder {
		return httptest.NewRecorder()
	}

	// move through handlers in reverse and compose them on top of each other
	for i := len(handlers) - 1; i >= 0; i-- {
		handler := handlers[i]
		next := chain
		chain = func(r *http.Request) *httptest.ResponseRecorder {
			return handler(r, next)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		recorder := chain(r)
		copyHeaders(recorder.Header(), w.Header())
		w.WriteHeader(recorder.Code)
		w.Write(recorder.Body.Bytes())
	}
}
