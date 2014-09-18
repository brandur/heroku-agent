package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

type SecondFactor struct {
	expiresAt time.Time
	token     string
	waiting   bool
}

var (
	secondFactors map[string]*SecondFactor
)

func init() {
	secondFactors = make(map[string]*SecondFactor)
}

func TwoFactorHandler(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder {
	auth := r.Header.Get("Authorization")

	if secondFactor, ok := secondFactors[auth]; ok {
		sentToken := r.Header.Get("Heroku-Two-Factor-Code")
		if secondFactor.token != "" {
			if secondFactor.expiresAt.After(time.Now()) {
				r.Header.Set("Authorization", "Bearer "+secondFactor.token)
				fmt.Printf("2FA token held; replaced authorization\n")
			} else {
				delete(secondFactors, auth)
				fmt.Printf("2FA token expired; removed from cache\n")
			}
		} else if secondFactor.waiting && sentToken != "" {
			// instead of just burning this token, request a specialized one
			// that can skip two factor checks, and which we'll hold onto
			token, expiresAt := getSkipTwoFactorToken(auth, sentToken)
			secondFactor.expiresAt = expiresAt
			secondFactor.token = token
			secondFactor.waiting = false
			fmt.Printf("2FA token acquired; set in cache\n")
		}
	}

	w := next(r)

	if w.Header().Get("Heroku-Two-Factor-Required") != "" {
		secondFactors[auth] = &SecondFactor{
			waiting: true,
		}
		fmt.Printf("2FA required; waiting on code from client\n")
	}

	return w
}

func getSkipTwoFactorToken(auth string, sentToken string) (string, time.Time) {
	return "", time.Now()
}
