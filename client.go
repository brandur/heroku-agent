package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
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
	logger.Printf("[client] Request: %s %s [start]\n", r.Method, safeUrl(r.URL))

	// unfortunately, we have to skip SSL verification on herokudev domains to
	// make things work because they are not CA-signed
	if isHerokuDev(r.Host) {
		t.transport.TLSClientConfig.InsecureSkipVerify = true
		defer func() {
			t.transport.TLSClientConfig.InsecureSkipVerify = false
		}()
	}

	resp, err := t.transport.RoundTrip(r)

	// only try to procure a status code if the request succeeded
	status := ""
	if err == nil {
		status = fmt.Sprintf(" [status=%v]", resp.StatusCode)
	}

	logger.Printf("[client] Response: %s %s [finish] [elapsed=%v]%v\n",
		r.Method, safeUrl(r.URL), time.Now().Sub(start), status)

	return resp, err
}

func init() {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: 1 * time.Minute,
		}).Dial,
		MaxIdleConnsPerHost: 5,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}
	client = &http.Client{
		// More aggressive timeout to minimize waits on a bad connection
		Timeout: 10 * time.Second,
		Transport: &InstrumentedTransport{
			transport: transport,
		},
	}
}

func DoRequest(r *http.Request) (*http.Response, error) {
	return client.Do(r)
}

func isHerokuDev(host string) bool {
	if strings.HasSuffix(host, ".herokudev.com") {
		return true
	}

	if strings.HasSuffix(host, ".herokudev.com:443") {
		return true
	}

	return false
}
