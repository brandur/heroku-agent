package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

var (
	client *http.Client
)

func ProxyHandler(w *httptest.ResponseRecorder, r *http.Request, next NextHandlerFunc) {
	url := "https://" + r.Host + r.URL.String()
	req, err := http.NewRequest(r.Method, url, r.Body)

	copyHeaders(r.Header, req.Header)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	next(w, r)

	for h, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Set(h, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	w.Write(bytes)
}

func copyHeaders(source http.Header, destination http.Header) {
	for h, vs := range source {
		for _, v := range vs {
			destination.Set(h, v)
		}
	}
}
