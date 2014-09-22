package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

type SecondFactor struct {
	expiresAt time.Time
	token     string
}

var (
	store *TwoFactorStore
)

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

type TwoFactorStore struct {
	secondFactorMap map[string]*SecondFactor
	mutex           *sync.Mutex
}

func init() {
	store = &TwoFactorStore{
		secondFactorMap: make(map[string]*SecondFactor),
		mutex:           &sync.Mutex{},
	}
}

func ReapTwoFactorStore() {
	for {
		select {
		case <-time.After(20 * time.Minute):
			store.reap()
		}
	}
}

func TwoFactorHandler(r *http.Request, next NextHandlerFunc) (*httptest.ResponseRecorder, error) {
	// replace our sent authorization if we're holding a more privileged token
	// already
	if !store.tryStoredSecondFactor(r) {
		// If a code was sent up, instead of just burning it, request a
		// specialized one that can skip two factor checks which we'll hold
		// onto. Don't do this if the user is trying to login because they
		// don't yet have any valid authorization.
		sentToken := r.Header.Get("Heroku-Two-Factor-Code")
		if sentToken != "" && !isLogin(r) {
			secondFactor, err := store.getSkipTwoFactorToken(r)
			if err != nil {
				return nil, err
			}
			store.setSecondFactor(r, secondFactor)

			// give the newly stored second factor another try
			store.tryStoredSecondFactor(r)
		}
	}

	return next(r)
}

func isLogin(r *http.Request) bool {
	return r.URL.Path == "/login"
}

func (s *TwoFactorStore) getSkipTwoFactorToken(r *http.Request) (*SecondFactor, error) {
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

	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

func (s *TwoFactorStore) reap() {
	numKeys := len(s.secondFactorMap)
	now := time.Now()
	expiredKeys := make([]string, 0)

	for k, v := range s.secondFactorMap {
		if now.After(v.expiresAt) {
			expiredKeys = append(expiredKeys, k)
		}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, k := range expiredKeys {
		delete(s.secondFactorMap, k)
	}

	logger.Printf("[2fa] Reaped %v of %v second factor(s)\n",
		len(expiredKeys), numKeys)
}

func (s *TwoFactorStore) setSecondFactor(r *http.Request, secondFactor *SecondFactor) {
	auth := r.Header.Get("Authorization")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.secondFactorMap[auth] = secondFactor
	logger.Printf("[2fa] 2FA token acquired; set in cache\n")
}

func (s *TwoFactorStore) tryStoredSecondFactor(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	secondFactor, ok := s.secondFactorMap[auth]

	if ok {
		if secondFactor.expiresAt.After(time.Now()) {
			r.Header.Set("Authorization", "Bearer "+secondFactor.token)
			logger.Printf("[2fa] 2FA token held; replaced authorization (valid for %v)\n",
				secondFactor.expiresAt.Sub(time.Now()))
			return true
		} else {
			delete(s.secondFactorMap, auth)
			logger.Printf("[2fa] 2FA token expired; removed from cache\n")
		}
	} else {
		logger.Printf("[2fa] 2FA token not held\n")
	}

	return false
}
