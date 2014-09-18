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
}

var (
	secondFactors map[string]*SecondFactor
)

func init() {
	secondFactors = make(map[string]*SecondFactor)
}

func tryStoredSecondFactor(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	secondFactor, ok := secondFactors[auth]

	if ok {
		if secondFactor.expiresAt.After(time.Now()) {
			r.Header.Set("Authorization", "Bearer "+secondFactor.token)
			fmt.Printf("2FA token held; replaced authorization (valid for %v)\n",
				secondFactor.expiresAt.Sub(time.Now()))
			return true
		} else {
			delete(secondFactors, auth)
			fmt.Printf("2FA token expired; removed from cache\n")
		}
	} else {
		fmt.Printf("2FA token not held\n")
	}

	return false
}

func TwoFactorHandler(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder {
	// replace our sent authorization if we're holding a more privileged token
	// already
	if !tryStoredSecondFactor(r) {
		sentToken := r.Header.Get("Heroku-Two-Factor-Code")
		if sentToken != "" {
			// instead of just burning this token, request a specialized one
			// that can skip two factor checks, and which we'll hold onto
			var err error
			secondFactor, err := getSkipTwoFactorToken(r)
			if err != nil {
				panic(err)
			}

			auth := r.Header.Get("Authorization")
			secondFactors[auth] = secondFactor
			fmt.Printf("2FA token acquired; set in cache\n")

			// give the newly stored second factor another try
			tryStoredSecondFactor(r)
		}
	}

	w := next(r)
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

func getSkipTwoFactorToken(r *http.Request) (*SecondFactor, error) {
	authUrl := "https://" + r.Host + "/oauth/authorizations"
	auth := r.Header.Get("Authorization")
	sentToken := r.Header.Get("Heroku-Two-Factor-Code")

	requestData := &CreateAuthorizationRequest{
		Description:   "heroku-agent",
		ExpiresIn:     60 * 30,
		SkipTwoFactor: true,
	}
	encoded, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", authUrl, bytes.NewBuffer(encoded))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Heroku-Two-Factor-Code", sentToken)
	req.Header.Set("X-Heroku-Sudo", "true")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("Unexpected response code: %v", resp.StatusCode)
	}

	encoded, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	responseData := &CreateAuthorizationResponse{}
	err = json.Unmarshal(encoded, responseData)
	if err != nil {
		return nil, err
	}

	secondFactor := &SecondFactor{
		expiresAt: time.Now().Add(time.Duration(responseData.AccessToken.ExpiresIn) * time.Second),
		token:     responseData.AccessToken.Token,
	}
	return secondFactor, nil
}
