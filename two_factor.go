package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
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

func ClearTwoFactorStore() {
	store.clear()
}

func ReapTwoFactorStore() {
	for {
		select {
		case <-time.After(20 * time.Minute):
			store.reap()
		}
	}
}

func TwoFactorStoreCount() int {
	return store.count()
}

func TwoFactorHandler(r *http.Request, next NextHandlerFunc) (*httptest.ResponseRecorder, error) {
	// replace our sent authorization if we're holding a more privileged token
	// already
	if !store.tryStoredSecondFactor(r) {
		// If a code was sent up, instead of just burning it, request a
		// specialized one that can skip two factor checks which we'll hold
		// onto. Don't do this if the user is trying to login because they
		// don't yet have any valid authorization.
		auth := r.Header.Get("Authorization")
		sentToken := r.Header.Get("Heroku-Two-Factor-Code")
		if hasAuth(auth) && sentToken != "" {
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

func hasAuth(auth string) bool {
	// "Og==" is just a colon ":" encoded in base64 (no user/pass)
	return auth != "" && !strings.HasSuffix(auth, "Og==")
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

func (s *TwoFactorStore) clear() {
	numKeys := len(s.secondFactorMap)
	for k := range s.secondFactorMap {
		delete(s.secondFactorMap, k)
	}
	logger.Printf("[2fa] Cleared %v second factor(s)\n", numKeys)
}

func (s *TwoFactorStore) count() int {
	return len(s.secondFactorMap)
}

// Normalizes the `Authorization` header.
//
// This isn't strictly necessary, but the API has a number of authentication
// techniques that will work against it, and trying to make them as uniform as
// possible helps to consolidate requests across clients that might be talking
// to heroku-agent.
func (s *TwoFactorStore) normalizeAuth(rawAuth string) string {
	// First of all, check for:
	//
	//     Authorization: Bearer <token>
	//
	// If we find it, then the normalized auth is the token itself.
	token := strings.TrimPrefix(rawAuth, "Bearer ")
	if token != rawAuth {
		return token
	}

	// See if we have a "Basic" authorization:
	//
	//     Authorization: Basic <base64 encoded creds>
	//
	// If we don't, then we don't know how to normalize this authorization, so
	// just return the opaque value.
	encodedAuth := strings.TrimPrefix(rawAuth, "Basic ")
	if encodedAuth == rawAuth {
		return rawAuth
	}

	decodedAuth, err := base64.StdEncoding.DecodeString(encodedAuth)
	if err != nil {
		return rawAuth
	}

	// See if we have a basic authorization with an empty user and a token:
	//
	//     Authorization: Basic <base64 encoded ":<token>">
	//
	// If we don't, then we probably have an "<email>:<token>" or
	// "<email>:<password>", which we shouldn't provide any special handling
	// for, so return the opaque value. We would theoretically like to handle
	// the former case, but unfortunately there's no way to differentiate
	// between the two.
	creds := strings.Split(string(decodedAuth), ":")
	if len(creds) != 2 || creds[0] != "" {
		return rawAuth
	}

	return creds[1]
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
	auth := s.normalizeAuth(r.Header.Get("Authorization"))
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.secondFactorMap[auth] = secondFactor
	logger.Printf("[2fa] 2FA token acquired; set in cache\n")
}

func (s *TwoFactorStore) tryStoredSecondFactor(r *http.Request) bool {
	auth := s.normalizeAuth(r.Header.Get("Authorization"))
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
