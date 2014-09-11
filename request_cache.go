package main

import (
	"fmt"
	"net/http"
)

var (
	contentHeaders map[string]bool
)

type CachedResponse struct {
	content []byte
	etag    string
	header  http.Header
}

type RequestCache struct {
	cacheMap map[string]map[string]*CachedResponse
}

func init() {
	contentHeaders = map[string]bool{
		"Content-Encoding": true,
		"Content-Length":   true,
		"Content-Type":     true,
		"Status":           true,
	}
}

func newRequestCache() *RequestCache {
	return &RequestCache{
		cacheMap: make(map[string]map[string]*CachedResponse),
	}
}

func (c *RequestCache) getCache(request *http.Request) (*CachedResponse, bool) {
	if request.Method != "GET" {
		return nil, false
	}

	auths, ok := request.Header["Authorization"]
	if !ok {
		return nil, false
	}

	url := request.URL.String()
	cached, ok := c.cacheMap[auths[0]][url]
	if !ok {
		fmt.Printf("cache miss: %s %s\n", auths[0][0:10], url)
		return nil, false
	}

	fmt.Printf("cache hit: %s %s (etag: %s)\n", auths[0][0:10], url, cached.etag)
	return cached, true
}

func (c *RequestCache) setCache(request *http.Request, response *http.Response, content []byte) {
	auths, ok := request.Header["Authorization"]
	if !ok {
		return
	}

	etags, ok := response.Header["Etag"]
	if !ok {
		return
	}

	c.cacheMap[auths[0]] = make(map[string]*CachedResponse)

	etag := etags[0]
	url := request.URL.String()
	cached := &CachedResponse{
		content: content,
		header:  make(http.Header),
		etag:    etag,
	}

	// store Content-* headers for an accurate cached response
	for h, vs := range response.Header {
		for _, v := range vs {
			if _, ok := contentHeaders[h]; ok {
				cached.header.Set(h, v)
			}
		}
	}

	c.cacheMap[auths[0]][url] = cached

	fmt.Printf("cache store: %s %s (etag: %s)\n", auths[0][0:10], url, etag)

	// @todo: check to make sure cache size doesn't become unmanagable
}
