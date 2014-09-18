package main

import (
	"net/http"
	"net/http/httptest"
)

func TwoFactorHandler(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder {
	return next(r)
}
