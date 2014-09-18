package main

import (
	"net/http"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	next(w, r)
}
