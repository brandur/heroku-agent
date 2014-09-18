package main

import (
	"io"
	"net/http"
	"net/http/httptest"
)

var (
	client *http.Client
)

func ProxyHandler(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder {
	url := "https://" + r.Host + r.URL.String()
	req, err := http.NewRequest(r.Method, url, r.Body)

	copyHeaders(r.Header, req.Header)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	w := next(r)

	for h, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Set(h, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		panic(err)
	}

	return w
}

func copyHeaders(source http.Header, destination http.Header) {
	for h, vs := range source {
		for _, v := range vs {
			destination.Set(h, v)
		}
	}
}
