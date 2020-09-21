package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (

	// defaultTimeout is the default amount of time to allow for all Clockify API calls to take place.
	defaultTimeout = time.Second * 10
)

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

	// Make requests starting last week at 0000h and end yesterday at 2400h.
	var loc *time.Location
	var err error
	if loc, err = time.LoadLocation("America/New_York"); err != nil {
		l.Fatalln(err.Error())
	}
	end := time.Now().In(loc).Truncate(time.Hour * 24)
	start := end.AddDate(0, 0, -7)

	// Make an HTTP client.
	client := &http.Client{}

	// Create a context.
	ctx, _ := context.WithTimeout(context.Background(), defaultTimeout)

	// Get an authentication token from Clockify.
	token := ""
	if token, err = authToken(ctx, client, clockifyEmail, clockifyPassword); err != nil {
		l.Fatalln(err.Error())
	}

	// Get the first workspace.
	workspace := ""
	if workspace, err = firstWorkspace(ctx, client, token); err != nil {
		l.Fatalln(err.Error())
	}

	// Get the PDF report.
	var pdfBytes []byte
	if pdfBytes, err = pdf(ctx, client, end, start, token, workspace); err != nil {
		l.Fatalln(err.Error())
	}

	// Get the total amount billable as a string.
	billable := ""
	sendBill := false
	if billable, sendBill, err = billTotal(ctx, client, end, end, token, workspace); err != nil {
		l.Fatalln(err.Error())
	}

	// Check to see if the bill should be sent.
	if !sendBill {
		l.Println("Not sending because there is nothing to bill.")
		return
	}

	// Make the email.
	body, subject := makeEmail(billable, start)

	// Send the email.
	if err = sendEmail([]byte(body), fromEmail, pdfBytes, smtpHost, smtpPassword, subject, to); err != nil {
		l.Fatalln(err.Error())
	}
}
