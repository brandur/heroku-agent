package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

const (
	NumRetries = 2
)

var (
	client *http.Client
)

func ProxyHandler(r *http.Request, next NextHandlerFunc) (*httptest.ResponseRecorder, error) {
	retries := NumRetries

retry:
	w, err := next(r)
	if err != nil {
		return w, err
	}

	url := "https://" + r.Host + r.URL.String()
	req, err := http.NewRequest(r.Method, url, r.Body)

	copyHeaders(r.Header, req.Header)

	resp, err := client.Do(req)
	if err != nil {
		// retry if this looks like this might be a temporary outage
		if shouldRetry(err) && retries > 0 {
			retries -= 1
			goto retry
		}

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

func shouldRetry(err error) bool {
	return strings.Contains(err.Error(), "connection reset by peer")
}
