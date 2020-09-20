package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/jordan-wright/email"
)

const (

	// billingEndpoint is the endpoint to reach out to to get the summary that has billing information.
	billingEndpoint = "https://reports.api.clockify.me/workspaces/%s/reports/summary"

	// defaultTimeout is the default amount of time to allow for all Clockify API calls to take place.
	defaultTimeout = time.Second * 10

	// billReqBody is the body of the response to send when requesting a summary for billing.
	billReqBody = `{
  "dateRangeStart": "%s",
  "dateRangeEnd": "%s",
  "sortOrder": "ASCENDING",
  "description": "",
  "rounding": false,
  "withoutDescription": false,
  "amountShown": "EARNED",
  "zoomLevel": "WEEK",
  "userLocale": "en_US",
  "customFields": null,
  "summaryFilter": {
    "sortColumn": "GROUP",
    "groups": [
      "PROJECT",
      "TIMEENTRY"
    ]
  }
}`

	// pdfReqBody is the body of the response to send when requesting a PDF summary.
	pdfReqBody = `{
  "dateRangeStart": "%s",
  "dateRangeEnd": "%s",
  "sortOrder": "ASCENDING",
  "description": "",
  "rounding": false,
  "withoutDescription": false,
  "amountShown": "EARNED",
  "zoomLevel": "WEEK",
  "userLocale": "en_US",
  "customFields": null,
  "summaryFilter": {
    "sortColumn": "GROUP",
    "groups": [
      "PROJECT",
      "TIMEENTRY"
    ]
  },
  "exportType": "PDF"
}`

	// pdfEndpoint is the endpoint to reach out to to get a PDF report.
	pdfEndpoint = "https://reports.api.clockify.me/workspaces/%s/reports/summary"

	// tokenEndpoint is the endpoint to reach out to to get an auth token.
	tokenEndpoint = "https://global.api.clockify.me/auth/token"

	// workspaceEndpoint is the endpoint to reach out to to get the user's workspaces.
	workspaceEndpoint = "https://global.api.clockify.me/workspaces/"
)

var (

	// errNoWorkspaces indicates that Clockify did not report any workspaces for this user.
	errNoWorkspaces = errors.New("no workspaces were found")
)

// billableResponse is the response from the Clockify API containing the info from a request for a summary.
type billableResponse struct {
	Totals []totalResponse `json:"totals"`
}

// credentials holds the Clockify credentails.
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// tokenResponse is the response from the Clockify API containing the requested auth token.
type tokenResponse struct {
	Token string `json:"token"`
}

// workspaceResponse is the response from the Clockify API containing the user's workspaces.
type workspaceResponse struct {
	Memberships []membershipsResponse `json:"memberships"`
}

// membershipsResponse is the response from the Clockify API containing the user's memberships.
type membershipsResponse struct {
	TargetId string `json:"targetId"`
}

// totalResponse is the response from the Clockify API containing the total billable amount.
type totalResponse struct {
	TotalAmount float64 `json:"totalAmount"`
}

// addTokenHeader adds the Clockify API token to the request header.
func addTokenHeader(req *http.Request, token string) {
	req.Header.Add("X-Auth-Token", token)
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

// billTotal gets the total amount of billable dollars from the Clockify API.
func billTotal(ctx context.Context, client *http.Client, now time.Time, token, workspace string) (billable string, sendBill bool, err error) {

	// Create the URL.
	url := fmt.Sprintf(billingEndpoint, workspace)

	// Start last week at 0000h and end yesterday at 2400h.
	var loc *time.Location
	if loc, err = time.LoadLocation("America/New_York"); err != nil {
		return "", false, err
	}
	now = time.Now().In(loc).Truncate(time.Hour * 24)
	lastWeek := now.AddDate(0, 0, -7)

	// Create the body as a string.
	bodyStr := fmt.Sprintf(billReqBody, lastWeek.Format("2006-01-02T15:04:05Z"), now.Format("2006-01-02T15:04:05Z"))

	// Create the request.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(bodyStr)); err != nil {
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
	var body []byte
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

// sendEmail sends the email with the report attachment.
func sendEmail(body []byte, from string, pdf []byte, smtpAddr, smtpPassword, subject string, to []string) (err error) {

	// Create the email.
	e := &email.Email{
		From:    from,
		To:      to,
		Subject: subject,
		Text:    body,
	}

	// Attach the PDF.
	if _, err = e.Attach(bytes.NewReader(pdf), "report.pdf", "application/pdf"); err != nil {
		return err
	}

	// Authenticate to the server.
	auth := smtp.PlainAuth("", from, smtpPassword, smtpAddr)

	// Send the email.
	if err = e.Send(smtpAddr+":587", auth); err != nil {
		return err
	}

	return nil
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

// pdf gets the PDF report from the Clockify API.
func pdf(ctx context.Context, client *http.Client, token, workspace string) (lastWeekStr string, pdfBytes []byte, now time.Time, err error) {

	// Start last week at 0000h and end yesterday at 2400h.
	var loc *time.Location
	if loc, err = time.LoadLocation("America/New_York"); err != nil {
		return "", nil, now, err
	}
	now = time.Now().In(loc).Truncate(time.Hour * 24)
	lastWeek := now.AddDate(0, 0, -7)
	lastWeekStr = fmt.Sprintf("%d-%d-%d", lastWeek.Year(), lastWeek.Month(), lastWeek.Day())

	// Create the URL.
	url := fmt.Sprintf(pdfEndpoint, workspace)

	// Create the body as a string.
	bodyStr := fmt.Sprintf(pdfReqBody, lastWeek.Format("2006-01-02T15:04:05Z"), now.Format("2006-01-02T15:04:05Z"))

	// Create the request.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(bodyStr)); err != nil {
		return lastWeekStr, nil, now, err
	}

	// Set the headers for the request.
	addTokenHeader(req, token)
	jsonHeader(req)

	// Set the URL query.
	req.URL.Query().Add("export", "pdf")

	// Perform the request.
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return lastWeekStr, nil, now, err
	}
	defer resp.Body.Close()

	// Get the body of the response.
	if pdfBytes, err = ioutil.ReadAll(resp.Body); err != nil {
		return lastWeekStr, nil, now, err
	}

	return lastWeekStr, pdfBytes, now, nil
}

// jsonHeader adds the JSON header to the request.
func jsonHeader(req *http.Request) {
	req.Header.Add("Content-Type", "application/json")
}

func main() {

	// Create a logger.
	l := log.New(os.Stdout, "cwrs: ", log.LstdFlags|log.Lshortfile)

	// Grab the environment variables.
	clockifyEmail := os.Getenv("CLOCKIFY_EMAIL")
	clockifyPassword := os.Getenv("CLOCKIFY_PASSWORD")
	fromEmail := os.Getenv("FROM_EMAIL")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	toEmails := os.Getenv("TO_EMAILS")
	for _, envVar := range []string{clockifyEmail, clockifyPassword, fromEmail, smtpHost, smtpPassword, toEmails} {
		if envVar == "" {
			l.Fatalln("Required environment variable empty.")
		}
	}

	// Build the destination emails.
	to := make([]string, 0)
	for _, emailStr := range strings.Split(toEmails, ",") {
		to = append(to, strings.TrimSpace(emailStr))
	}
	if len(to) == 0 {
		l.Fatalln("No destination emails were set.")
	}

	// Make an HTTP client.
	client := &http.Client{}

	// Create a context.
	ctx, _ := context.WithTimeout(context.Background(), defaultTimeout)

	// Get an authentication token from Clockify.
	token := ""
	var err error
	if token, err = authToken(ctx, client, clockifyEmail, clockifyPassword); err != nil {
		l.Fatalln(err.Error())
	}

	// Get the first workspace.
	workspace := ""
	if workspace, err = firstWorkspace(ctx, client, token); err != nil {
		l.Fatalln(err.Error())
	}

	// Get the PDF report.
	lastWeek := ""
	var now time.Time
	var pdfBytes []byte
	if lastWeek, pdfBytes, now, err = pdf(ctx, client, token, workspace); err != nil {
		l.Fatalln(err.Error())
	}

	// Get the total amount billable as a string.
	billable := ""
	sendBill := false
	if billable, sendBill, err = billTotal(ctx, client, now, token, workspace); err != nil {
		l.Fatalln(err.Error())
	}

	// Check to see if the bill should be sent.
	if !sendBill {
		l.Println("Not sending because there is nothing to bill.")
		return
	}

	// Make the email.
	body, subject := makeEmail(billable, lastWeek)

	// Send the email.
	if err = sendEmail([]byte(body), fromEmail, pdfBytes, smtpHost, smtpPassword, subject, to); err != nil {
		l.Fatalln(err.Error())
	}
}

// makeEmail creates the body and the subject of the email.
func makeEmail(bill, lastWeek string) (body, subject string) {
	body = fmt.Sprintf("Attached you will find the weekly report for %s.\n\nThe total for the week is: "+
		"%s. Please validate this with the attached report.\n\n\nbeep boop.\nThis is an automated email set for every "+
		"Monday at 0400 EST.\n\nThis email is not monitored.", lastWeek, bill)
	subject = fmt.Sprintf("%s Weekly Report (AUTOMATED)", lastWeek)
	return body, subject
}
