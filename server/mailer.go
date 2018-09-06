package server

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"sync"
)

// Mailer is a mail sender.
type Mailer interface {
	Send(to *mail.Address, data interface{}) error
}

// SMTPMailer uses net/smtp to send an email.
type SMTPMailer struct {
	Address  string
	Username string
	Password string
	From     *mail.Address
	Subject  *template.Template
	Body     *template.Template
}

// Send sends the email to the specified address.
func (s *SMTPMailer) Send(to *mail.Address, data interface{}) error {
	conn, err := smtp.Dial(s.Address)
	if err != nil {
		return err
	}
	host, _, _ := net.SplitHostPort(s.Address)
	if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: true, ServerName: host}); err != nil {
		return err
	}
	if err := conn.Auth(smtp.PlainAuth("", s.Username, s.Password, host)); err != nil {
		return err
	}
	if err := conn.Mail(s.From.Address); err != nil {
		return err
	}
	if err := conn.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := conn.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	if err = s.writeMessage(w, to, data); err != nil {
		return err
	}
	if err := conn.Quit(); err != nil {
		terr, ok := err.(*textproto.Error)
		if !ok || terr.Code != 250 {
			return err
		}
	}
	return nil
}

func (s *SMTPMailer) writeMessage(w io.Writer, to *mail.Address, data interface{}) error {
	b := getbuffer()
	defer releaseBuffer(b)
	if err := s.Subject.Execute(b, data); err != nil {
		return err
	}
	headers := map[string]string{
		"From":         s.From.String(),
		"To":           to.String(),
		"Subject":      b.String(),
		"Content-Type": "text/html; charset=utf-8",
	}
	for k, v := range headers {
		fmt.Fprintf(w, "%s: %s\r\n", k, v)
	}
	fmt.Fprint(w, "\r\n")
	return s.Body.Execute(w, data)
}

var buffers = sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}

func getbuffer() *bytes.Buffer        { return buffers.Get().(*bytes.Buffer) }
func releaseBuffer(buf *bytes.Buffer) { buf.Reset(); buffers.Put(buf) }
