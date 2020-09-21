package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (

	// billingEndpoint is the endpoint to reach out to to get the summary that has billing information.
	billingEndpoint = "https://reports.api.clockify.me/workspaces/%s/reports/summary"
)

// billableResponse is the response from the Clockify API containing the info from a request for a summary.
type billableResponse struct {
	Totals []totalResponse `json:"totals"`
}

// totalResponse is the response from the Clockify API containing the total billable amount.
type totalResponse struct {
	TotalAmount float64 `json:"totalAmount"`
}

// billTotal gets the total amount of billable dollars from the Clockify API.
func billTotal(ctx context.Context, client *http.Client, end, start time.Time, token, workspace string) (billable string, sendBill bool, err error) {

	// Create the URL.
	url := fmt.Sprintf(billingEndpoint, workspace)

	// Create the body as a string.
	var body []byte
	if body, err = makeBody(end, "", start); err != nil {
		return "", false, err
	}

	// Create the request.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body)); err != nil {
		return "", false, err
	}

	// Set the headers for the request.
	addTokenHeader(req, token)
	jsonHeader(req)

	// Perform the request.
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	// Get the body of the response.
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", false, err
	}

	// Unmarshal the body into the expected structure.
	bill := &billableResponse{}
	if err = json.Unmarshal(body, bill); err != nil {
		return "", false, err
	}

	// Make sure there is something to charge.
	if len(bill.Totals) == 0 {
		return "", false, nil
	}

	// Make sure there is something to charge.
	if bill.Totals[0].TotalAmount == 0 {
		return "", false, nil
	}

	// Get the total in a dollar amount.
	total := bill.Totals[0].TotalAmount / 100

	return "$" + fmt.Sprintf("%.2f", total), true, err
}
