package main

import (
	"bytes"
	"net/smtp"

	"github.com/jordan-wright/email"
)

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
