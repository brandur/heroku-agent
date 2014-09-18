package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
			token, expiresAt, err := getSkipTwoFactorToken(r.URL.String(), auth, sentToken)
			if err != nil {
				panic(err)
			}

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

type CreateAuthorizationRequest struct {
	Description   string `json:"description"`
	ExpiresIn     int    `json:"expires_in"`
	SkipTwoFactor bool   `json:"skip_two_factor"`
}

type CreateAuthorizationResponse struct {
	AccessToken struct {
		ExpiresIn int    `json:"expires_in"`
		Token     string `json:"token"`
	} `json:"access_token"`
}

func getSkipTwoFactorToken(url string, auth string, sentToken string) (string, time.Time, error) {
	fmt.Printf("url = %s\n", url)

	requestData := &CreateAuthorizationRequest{
		Description:   "Skip 2FA session from heroku-agent",
		ExpiresIn:     60 * 30,
		SkipTwoFactor: true,
	}
	encoded, err := json.Marshal(requestData)
	if err != nil {
		return "", time.Now(), err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(encoded))
	if err != nil {
		return "", time.Now(), err
	}

	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Heroku-Sudo", "true")

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Now(), err
	}
	if resp.StatusCode != 201 {
		return "", time.Now(), fmt.Errorf("Unexpected response code: %v", resp.StatusCode)
	}

	encoded, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", time.Now(), err
	}

	responseData := &CreateAuthorizationResponse{}
	err = json.Unmarshal(encoded, responseData)
	if err != nil {
		return "", time.Now(), err
	}

	return responseData.AccessToken.Token, time.Now().Add(time.Duration(responseData.AccessToken.ExpiresIn) * time.Second), nil
}
