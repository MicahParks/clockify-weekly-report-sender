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
	billingEndpoint = "https://global.api.clockify.me/workspaces/%s/reports/new/summary/"
	defaultTimeout  = time.Second * 10
	pdfReqBody      = `{
    "userGroupIds": [],
    "userIds": [],
    "projectIds": [],
    "clientIds": [
      "5cfd9075a02f7a6dc1bba9c7"
    ],
    "taskIds": [],
    "tagIds": [],
    "billable": "BOTH",
    "description": "",
    "firstTime": true,
    "archived": "Active",
    "startDate": "%s",
    "endDate": "%s",
    "me": "TEAM",
    "includeTimeEntries": true,
    "zoomLevel": "week",
    "name": "",
    "groupingOn": true,
    "groupedByDate": false,
    "page": 0,
    "sortDetailedBy": "timeAsc",
    "count": 500,
    "roundingOn": false,
    "isDetailed": true,
    "groupBy": "PROJECT",
    "subgroupBy": "TIME_ENTRY",
    "weeklyGroupBy": "PROJECT",
    "weeklySubgroupBy": "TIME"
  }`
	pdfEndpoint       = "https://global.api.clockify.me/workspaces/%s/reports/summary"
	tokenEndpoint     = "https://global.api.clockify.me/auth/token"
	workspaceEndpoint = "https://global.api.clockify.me/workspaces/"
)

var (
	errNoWorkspaces = errors.New("no workspaces were found")
)

type billableResponse struct {
	TotalBillable int `json:"totalBillable"`
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type workspaceResponse struct {
	Memberships []membershipsResponse `json:"memberships"`
}

type membershipsResponse struct {
	TargetId string `json:"targetId"`
}

func addTokenHeader(req *http.Request, token string) {
	req.Header.Add("X-Auth-Token", token)
}

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
	jsonHeaders(req)

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

func billTotal(ctx context.Context, client *http.Client, reqBody []byte, token, workspace string) (billable string, sendBill bool, err error) {

	// Create the URL.
	url := fmt.Sprintf(billingEndpoint, workspace)

	// Create the request.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody)); err != nil {
		return "", false, err
	}

	// Set the headers for the request.
	addTokenHeader(req, token)
	jsonHeaders(req)

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

	if bill.TotalBillable == 0 {
		return "", false, nil
	}

	// Get the total in a dollar amount.
	total := float64(bill.TotalBillable) / 100

	return "$" + fmt.Sprintf("%.2f", total), true, err
}

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
	if err = e.Send(smtpAddr, auth); err != nil {
		return err
	}

	return nil
}

func firstWorkspace(ctx context.Context, client *http.Client, token string) (workspace string, err error) {

	// Create the request.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, workspaceEndpoint, bytes.NewReader(nil)); err != nil {
		return "", err
	}

	// Set the headers for the request.
	addTokenHeader(req, token)
	jsonHeaders(req)

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
	workspaces := []workspaceResponse{}
	if err = json.Unmarshal(body, workspaces); err != nil {
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

func pdf(ctx context.Context, client *http.Client, token, workspace string) (lastWeekStr string, pdfBytes, reqBody []byte, err error) {

	// Start last week at 0000h and end yesterday at 2400h.
	var loc *time.Location
	if loc, err = time.LoadLocation("America/New_York"); err != nil {
		return "", nil, nil, err
	}
	now := time.Now().In(loc).Truncate(time.Hour * 24)
	lastWeek := now.AddDate(0, 0, -7)
	lastWeekStr = fmt.Sprintf("%d-%d-%d", lastWeek.Year(), lastWeek.Month(), lastWeek.Day())

	// Create the URL.
	url := fmt.Sprintf(pdfEndpoint, workspace)

	// Create the body as a string.
	bodyStr := fmt.Sprintf(pdfReqBody, lastWeek.Format(time.RFC3339), now.Format(time.RFC3339))

	// Create the request.
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(bodyStr)); err != nil {
		return lastWeekStr, nil, nil, err
	}

	// Set the headers for the request.
	addTokenHeader(req, token)
	jsonHeaders(req)

	// Set the URL query.
	req.URL.Query().Add("export", "pdf")

	// Perform the request.
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return lastWeekStr, nil, nil, err
	}
	defer resp.Body.Close()

	// Get the body of the response.
	if pdfBytes, err = ioutil.ReadAll(resp.Body); err != nil {
		return lastWeekStr, nil, nil, err
	}

	return lastWeekStr, pdfBytes, []byte(bodyStr), nil
}

func jsonHeaders(req *http.Request) {
	req.Header.Add("Content-Type", "application/json")
}

func main() {

	// Create a logger.
	l := log.New(os.Stdout, "cwrs: ", log.LstdFlags|log.Lshortfile)

	// Grab the environment variables.
	clockifyEmail := os.Getenv("CLOCKIFY_EMAIL")
	clockifyPassword := os.Getenv("CLOCKIFY_PASSWORD")
	fromEmail := os.Getenv("FROM_ENV")
	smtpAddr := os.Getenv("SMTP_ADDR")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	toEmails := os.Getenv("TO_EMAILS")
	for _, envVar := range []string{clockifyEmail, clockifyPassword, fromEmail, smtpAddr, smtpPassword, to} {
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
	ctx, _ := defaultContext()

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
	var pdfBytes []byte
	var reqBody []byte
	if lastWeek, pdfBytes, reqBody, err = pdf(ctx, client, token, workspace); err != nil {
		l.Fatalln(err.Error())
	}

	// Get the total amount billable as a string.
	billable := ""
	sendBill := false
	if billable, sendBill, err = billTotal(ctx, client, reqBody, token, workspace); err != nil {
		l.Fatalln(err.Error())
	}

	// Check to see if the bill should be sent.
	if !sendBill {
		return
	}

	// Make the email.
	body, subject := makeEmail(billable, lastWeek)

	// Send the email.
	if err = sendEmail([]byte(body), fromEmail, pdfBytes, smtpAddr, smtpPassword, subject, to); err != nil {
		l.Fatalln(err.Error())
	}
}

func makeEmail(bill, lastWeek string) (body, subject string) {
	body = fmt.Sprintf("Attached you will find the weekly report for %s.\n\nThe total for the week is: "+
		"%s. Please validate this with the attached report.\n\n\nbeep boop.\nThis is an automated email set for every "+
		"Monday at 0400 EST. If you'd like to make a suggestion about when it should be sent, the content, if you "+
		"see a mistake, or if you have a suggestion, please reply to it.", lastWeek, bill)
	subject = fmt.Sprintf("Weekly Report for %s (AUTOMATED)", lastWeek)
	return body, subject
}

func defaultContext() (ctx context.Context, cancel context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}
