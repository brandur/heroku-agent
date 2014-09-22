package main

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	client *http.Client
)

type InstrumentedTransport struct {
	transport *http.Transport
}

func (t *InstrumentedTransport) CancelRequest(r *http.Request) {
	t.transport.CancelRequest(r)
}

func (t *InstrumentedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	logger.Printf("[client] Request: %s %s [start]\n", r.Method, r.URL.String())

	resp, err := t.transport.RoundTrip(r)

	// only try to procure a status code if the request succeeded
	status := ""
	if err == nil {
		status = fmt.Sprintf(" [status=%v]", resp.StatusCode)
	}

	logger.Printf("[client] Response: %s %s [finish] [elapsed=%v]%v\n",
		r.Method, r.URL.String(), time.Now().Sub(start), status)

	return resp, err
}

func init() {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: 1 * time.Minute,
		}).Dial,
		MaxIdleConnsPerHost: 5,
	}
	client = &http.Client{
		// More aggressive timeout to minimize waits on a bad connection
		Timeout: 10 * time.Second,
		Transport: &InstrumentedTransport{
			transport: transport,
		},
	}
}
