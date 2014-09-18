package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	cache          *RequestCache
	client         *http.Client
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

func CacheHandler(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	cached, isCached := cache.getCache(r)

	// don't try our cache is the client sent their own cache attempt
	if _, ok := r.Header["If-None-Match"]; ok {
		isCached = false
	}

	url := "https://" + r.Host + r.URL.String()
	req, err := http.NewRequest(r.Method, url, r.Body)

	if isCached {
		req.Header.Set("If-None-Match", cached.etag)
	}

	copyHeaders(r.Header, req.Header)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	for h, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Set(h, v)
		}
	}
	if isCached && resp.StatusCode == 304 {
		// remove headers that may be inaccurate on a cached response
		for k, _ := range contentHeaders {
			w.Header().Del(k)
		}
		copyHeaders(cached.header, w.Header())

		w.WriteHeader(200)
		w.Write(cached.content)
	} else {
		w.WriteHeader(resp.StatusCode)
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		cache.setCache(r, resp, bytes)
		w.Write(bytes)
	}

	fmt.Printf("served: %s %s (%v)\n", r.Method, r.URL.Path, resp.StatusCode)
}

func copyHeaders(source http.Header, destination http.Header) {
	for h, vs := range source {
		for _, v := range vs {
			destination.Set(h, v)
		}
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

	auth := auths[0]
	url := request.URL.String()
	cached, ok := c.cacheMap[auth][url]
	if !ok {
		fmt.Printf("cache miss: %s %s\n", auth[0:10], url)
		return nil, false
	}

	fmt.Printf("cache hit: %s %s (etag: %s)\n", auth[0:10], url, cached.etag)
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

	auth := auths[0]
	if _, ok = c.cacheMap[auth]; !ok {
		c.cacheMap[auths[0]] = make(map[string]*CachedResponse)
	}

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

	c.cacheMap[auth][url] = cached

	fmt.Printf("cache store: %s %s (etag: %s)\n", auth[0:10], url, etag)

	// @todo: check to make sure cache size doesn't become unmanagable
}
