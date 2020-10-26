package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (

	// tokenEndpoint is the endpoint to reach out to to get an auth token.
	tokenEndpoint = "https://global.api.clockify.me/auth/token"
)

// credentials holds the Clockify credentials.
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// tokenResponse is the response from the Clockify API containing the requested auth token.
type tokenResponse struct {
	Token string `json:"token"`
}

// authToken gets the Clockify API token by logging into the Clockify API.
func authToken(ctx context.Context, client *http.Client, email, password string) (authToken string, err error) {

	// Create the credentials structure.
	creds := &credentials{
		Email:    email,
		Password: password,
	}

	// Turn the credentials into bytes.
	var body []byte
	if body, err = json.Marshal(creds); err != nil {
		return "", err
	}

	// Create the request to get a token with.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, bytes.NewReader(body)); err != nil {
		return "", err
	}

	// Set the headers for the request.
	jsonHeader(req)

	// Perform the request.
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Get the body of the response.
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", err
	}

	// Unmarshal the body into the expected structure.
	token := &tokenResponse{}
	if err = json.Unmarshal(body, token); err != nil {
		return "", err
	}

	return token.Token, nil
}
