package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

var (
	cache          *RequestCache
	contentHeaders map[string]bool
)

type CachedResponse struct {
	content   []byte
	etag      string
	expiresAt time.Time
	header    http.Header
}

type RequestCache struct {
	cacheMap map[string]*CachedResponse
	mutex    *sync.Mutex
}

func init() {
	cache = &RequestCache{
		cacheMap: make(map[string]*CachedResponse),
		mutex:    &sync.Mutex{},
	}
	contentHeaders = map[string]bool{
		"Content-Encoding": true,
		"Content-Length":   true,
		"Content-Type":     true,
		"Status":           true,
	}
}

func CacheHandler(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder {
	cached, isCached := cache.getCache(r)

	// don't try our cache if the client sent their own cache attempt
	if _, ok := r.Header["If-None-Match"]; ok {
		isCached = false
	}

	if isCached {
		r.Header.Set("If-None-Match", cached.etag)
	}

	w := next(r)

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

	return w
}

func ReapCache() {
	for {
		select {
		case <-time.After(20 * time.Minute):
			cache.reap()
		}
	}
}

func (c *RequestCache) buildCacheKey(request *http.Request) string {
	auth := request.Header.Get("Authorization")
	url := request.URL.String()
	return auth + ":" + request.Method + ":" + request.Host + ":" + url
}

func (c *RequestCache) getCache(request *http.Request) (*CachedResponse, bool) {
	if request.Method != "GET" {
		return nil, false
	}

	auth := request.Header.Get("Authorization")
	if auth == "" {
		return nil, false
	}

	cached, ok := c.cacheMap[c.buildCacheKey(request)]
	if !ok {
		fmt.Printf("Cache miss: %s... %s%s\n",
			auth[0:10], request.Host, request.URL.String())

		return nil, false
	}

	fmt.Printf("Cache hit: %s... %s%s [etag=%s]\n",
		auth[0:10], request.Host, request.URL.String(), cached.etag)

	return cached, true
}

func (c *RequestCache) reap() {
	now := time.Now()
	expiredKeys := make([]string, 0)

	for k, v := range c.cacheMap {
		if now.After(v.expiresAt) {
			expiredKeys = append(expiredKeys, k)
		}
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, k := range expiredKeys {
		delete(c.cacheMap, k)
	}

	fmt.Printf("Reaped %v cache key(s) of %v\n",
		len(expiredKeys), len(c.cacheMap))
}

func (c *RequestCache) setCache(request *http.Request, headers http.Header, content []byte) {
	auth := request.Header.Get("Authorization")
	if auth == "" {
		return
	}

	etag := headers.Get("Etag")
	if etag == "" {
		return
	}

	url := request.URL.String()
	cached := &CachedResponse{
		content:   content,
		expiresAt: time.Now().Add(60 * time.Minute),
		header:    make(http.Header),
		etag:      etag,
	}

	// store Content-* headers for an accurate cached response
	for h, vs := range headers {
		for _, v := range vs {
			if _, ok := contentHeaders[h]; ok {
				cached.header.Set(h, v)
			}
		}
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cacheMap[c.buildCacheKey(request)] = cached

	fmt.Printf("Cache store: %s... %s%s [etag=%s]\n",
		auth[0:10], request.Host, url, etag)
}
