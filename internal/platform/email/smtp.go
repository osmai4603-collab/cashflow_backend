package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	SSL      bool
	From     string
}

type Sender interface {
	Send(to string, subject, body string) error
}

type SMTPSender struct {
	cfg Config
}

func NewSMTPSender(cfg Config) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(to string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	msg := fmt.Appendf(nil, "To: %s\r\nSubject: %s\r\n\r\n%s\r\n", to, subject, body)

	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)

	if s.cfg.SSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // For development, should be configurable
			ServerName:         s.cfg.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			return err
		}
		defer c.Quit()

		if err = c.Auth(auth); err != nil {
			return err
		}

		if err = c.Mail(s.cfg.From); err != nil {
			return err
		}

		if err = c.Rcpt(to); err != nil {
			return err
		}

		w, err := c.Data()
		if err != nil {
			return err
		}

		_, err = w.Write(msg)
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}

		return nil
	}

	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, msg)
}
