package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

const (
	NumRetries = 2
)

func ProxyHandler(r *http.Request, next NextHandlerFunc) (*httptest.ResponseRecorder, error) {
	retriesLeft := NumRetries

retry:
	w, err := next(r)
	if err != nil {
		return w, err
	}

	u := url.URL{
		Host:     r.Host,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
		Scheme:   "https",
	}

	req, err := http.NewRequest(r.Method, u.String(), r.Body)

	copyHeaders(r.Header, req.Header)

	resp, err := DoRequest(req)
	if err != nil {
		// retry if this looks like this might be a temporary outage
		if shouldRetry(err) && retriesLeft > 0 {
			retriesLeft -= 1
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
