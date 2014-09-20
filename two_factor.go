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
	store *SecondFactorStore
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

type SecondFactorStore struct {
	secondFactorMap map[string]*SecondFactor
	mutex           *sync.Mutex
}

func init() {
	store = &SecondFactorStore{
		secondFactorMap: make(map[string]*SecondFactor),
		mutex:           &sync.Mutex{},
	}
}

func ReapSecondFactorStore() {
	for {
		select {
		case <-time.After(20 * time.Minute):
			store.reap()
		}
	}
}

func TwoFactorHandler(r *http.Request, next NextHandlerFunc) *httptest.ResponseRecorder {
	// replace our sent authorization if we're holding a more privileged token
	// already
	if !store.tryStoredSecondFactor(r) {
		sentToken := r.Header.Get("Heroku-Two-Factor-Code")
		if sentToken != "" {
			// instead of just burning this token, request a specialized one
			// that can skip two factor checks, and which we'll hold onto
			secondFactor, err := store.getSkipTwoFactorToken(r)
			if err != nil {
				logger.Panic(err)
			}
			store.setSecondFactor(r, secondFactor)

			// give the newly stored second factor another try
			store.tryStoredSecondFactor(r)
		}
	}

	w := next(r)
	return w
}

func (s *SecondFactorStore) getSkipTwoFactorToken(r *http.Request) (*SecondFactor, error) {
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

	resp, err := client.Do(req)
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

func (s *SecondFactorStore) reap() {
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

func (s *SecondFactorStore) setSecondFactor(r *http.Request, secondFactor *SecondFactor) {
	auth := r.Header.Get("Authorization")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.secondFactorMap[auth] = secondFactor
	logger.Printf("[2fa] 2FA token acquired; set in cache\n")
}

func (s *SecondFactorStore) tryStoredSecondFactor(r *http.Request) bool {
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
