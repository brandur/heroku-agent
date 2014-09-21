package main

import (
	"io"
	"net/http"
	"net/http/httptest"
)

var (
	client *http.Client
)

func ProxyHandler(r *http.Request, next NextHandlerFunc) (*httptest.ResponseRecorder, error) {
	w, err := next(r)
	if err != nil {
		return w, err
	}

	url := "https://" + r.Host + r.URL.String()
	req, err := http.NewRequest(r.Method, url, r.Body)

	copyHeaders(r.Header, req.Header)

	resp, err := client.Do(req)
	if err != nil {
		return w, err
	}
	defer resp.Body.Close()

	copyHeaders(resp.Header, w.Header())

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return w, err
	}

	return w, nil
}
