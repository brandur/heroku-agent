package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

var (
	cache          *RequestCache
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

func CacheHandler(w *httptest.ResponseRecorder, r *http.Request, next NextHandlerFunc) {
	cached, isCached := cache.getCache(r)

	// don't try our cache if the client sent their own cache attempt
	if _, ok := r.Header["If-None-Match"]; ok {
		isCached = false
	}

	if isCached {
		r.Header.Set("If-None-Match", cached.etag)
	}

	next(w, r)

	if isCached && w.Code == 304 {
		newWriter := httptest.NewRecorder()

		// remove headers that may be inaccurate on a cached response
		for k, _ := range contentHeaders {
			w.Header().Del(k)
		}
		copyHeaders(w.Header(), newWriter.Header())
		copyHeaders(cached.header, newWriter.Header())

		newWriter.WriteHeader(200)
		newWriter.Write(cached.content)

		// move to the new writer reference and discard the old one
		w = newWriter
	} else {
		cache.setCache(r, w.Header(), w.Body.Bytes())
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

func (c *RequestCache) setCache(request *http.Request, headers http.Header, content []byte) {
	auths, ok := request.Header["Authorization"]
	if !ok {
		return
	}

	etags, ok := headers["Etag"]
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
	for h, vs := range headers {
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
