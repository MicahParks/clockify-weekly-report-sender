package main

import (
	"fmt"
	"net/http"
	"time"
)

// addTokenHeader adds the Clockify API token to the request header.
func addTokenHeader(req *http.Request, token string) {
	req.Header.Add("X-Auth-Token", token)
}

// bodySubject creates the body and the subject of the email.
func bodySubject(bill string, start time.Time) (body, subject string) {

	// Make the start date in human readable format.
	startDateStr := fmt.Sprintf("%d-%d-%d", start.Year(), start.Month(), start.Day())

	// Make the body and the subject.
	body = fmt.Sprintf("Attached you will find the weekly report for %s.\n\nThe total for the week is: "+
		"%s. Please validate this with the attached report.\n\n\nbeep boop.\nThis is an automated email set for every "+
		"Monday at 0400 EST.\n\nThis email is not monitored.", startDateStr, bill)
	subject = fmt.Sprintf("%s Weekly Report (AUTOMATED)", startDateStr)

	return body, subject
}

// jsonHeader adds the JSON header to the request.
func jsonHeader(req *http.Request) {
	req.Header.Add("Content-Type", "application/json")
}
