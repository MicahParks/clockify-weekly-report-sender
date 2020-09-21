package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

const (

	// workspaceEndpoint is the endpoint to reach out to to get the user's workspaces.
	workspaceEndpoint = "https://global.api.clockify.me/workspaces/"
)

var (

	// errNoWorkspaces indicates that Clockify did not report any workspaces for this user.
	errNoWorkspaces = errors.New("no workspaces were found")
)

// membershipsResponse is the response from the Clockify API containing the user's memberships.
type membershipsResponse struct {
	TargetId string `json:"targetId"`
}

// workspaceResponse is the response from the Clockify API containing the user's workspaces.
type workspaceResponse struct {
	Memberships []membershipsResponse `json:"memberships"`
}

// firstWorkspace gets the first workspace for the user from the Clockify API.
func firstWorkspace(ctx context.Context, client *http.Client, token string) (workspace string, err error) {

	// Create the request.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, workspaceEndpoint, bytes.NewReader(nil)); err != nil {
		return "", err
	}

	// Set the headers for the request.
	addTokenHeader(req, token)
	jsonHeader(req)

	// Perform the request.
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Get the body of the response.
	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", err
	}

	// Unmarshal the body into the expected structure.
	var workspaces []workspaceResponse
	if err = json.Unmarshal(body, &workspaces); err != nil {
		return "", err
	}

	// Get the first workspace's target ID.
	first := ""
	if len(workspaces) > 0 && len(workspaces[0].Memberships) > 0 {
		first = workspaces[0].Memberships[0].TargetId
	} else {
		return "", errNoWorkspaces
	}

	return first, nil
}
