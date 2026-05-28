package email

import (
	"fmt"
	"net/smtp"
)

type EmailSender interface {
	Send(to, subject, body string) error
}

type smtpSender struct {
	host   string
	port   int
	from   string
	user   string
	pass   string
	secure bool
}

func NewSMTPSender(host string, port int, from, user, pass string, secure bool) EmailSender {
	return &smtpSender{
		host:   host,
		port:   port,
		from:   from,
		user:   user,
		pass:   pass,
		secure: secure,
	}
}

func (s *smtpSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	msg := buildMessage(s.from, to, subject, body)

	if s.user == "" {
		return smtp.SendMail(addr, nil, s.from, []string{to}, msg)
	}

	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	return smtp.SendMail(addr, auth, s.from, []string{to}, msg)
}

func buildMessage(from, to, subject, body string) []byte {
	return []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body,
	))
}
