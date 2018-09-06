package server

import (
	"html/template"
	"net/mail"
	"os"
	"testing"
)

func TestMail(t *testing.T) {
	from, err := mail.ParseAddress(os.Getenv("EMAIL_FROM"))
	if err != nil {
		t.Fatal(err)
	}
	to, err := mail.ParseAddress(os.Getenv("EMAIL_TO"))
	if err != nil {
		t.Fatal(err)
	}
	subject, err := template.New("").Parse("Invitation to {{.organisation}}")
	if err != nil {
		t.Fatal(err)
	}
	body, err := template.New("").Parse(`You have been invited to <b>{{.organisation}}</b>, you secret code is <b>{{.secret}}</b>`)
	if err != nil {
		t.Fatal(err)
	}
	var mailer = SMTPMailer{
		Address:  os.Getenv("EMAIL_ADDRESS"),
		Username: os.Getenv("EMAIL_USERNAME"),
		Password: os.Getenv("EMAIL_PASSWORD"),
		From:     from,
		Subject:  subject,
		Body:     body,
	}

	if err := mailer.Send(to, map[string]string{"organisation": "banana", "secret": "SICRET!1"}); err != nil {
		t.Fatalf("%#v", err)
	}
}
