package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

type HerokuApiError struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

func ErrorHandler(r *http.Request, next NextHandlerFunc) (*httptest.ResponseRecorder, error) {
	w, err := next(r)

	if err != nil {
		logger.Printf("[error] %s\n", err.Error())
		apiError := &HerokuApiError{
			Id:      "heroku_agent",
			Message: "heroku-agent: " + err.Error(),
		}
		data, err := json.Marshal(*apiError)
		if err != nil {
			return w, err
		}
		w.WriteHeader(500)
		w.Write(data)
	}

	return w, nil
}
