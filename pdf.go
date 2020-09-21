package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (

	// pdfEndpoint is the endpoint to reach out to to get a PDF report.
	pdfEndpoint = "https://reports.api.clockify.me/workspaces/%s/reports/summary"
)

// pdf gets the PDF report from the Clockify API.
func pdf(ctx context.Context, client *http.Client, end, start time.Time, token, workspace string) (pdfBytes []byte, err error) {

	// Create the URL.
	url := fmt.Sprintf(pdfEndpoint, workspace)

	// Create the body as a string.
	var body []byte
	if body, err = makeBody(end, "PDF", start); err != nil {
		return nil, err
	}

	// Create the request.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body)); err != nil {
		return nil, err
	}

	// Set the headers for the request.
	addTokenHeader(req, token)
	jsonHeader(req)

	// Set the URL query.
	req.URL.Query().Add("export", "pdf")

	// Perform the request.
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Get the body of the response.
	if pdfBytes, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	return pdfBytes, nil
}
