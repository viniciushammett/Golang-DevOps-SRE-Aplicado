package notify

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/viniciushammett/go-alert-router/internal/config"
	"github.com/viniciushammett/go-alert-router/internal/logger"
)

type Email struct {
	log *logger.Logger
	cfg config.EmailConfig
}

func NewEmail(log *logger.Logger, cfg config.EmailConfig) *Email {
	return &Email{log: log, cfg: cfg}
}

func (e *Email) Send(to []string, subject, body string) error {
	if e.cfg.SMTPHost == "" { return fmt.Errorf("smtp not configured") }
	addr := fmt.Sprintf("%s:%d", e.cfg.SMTPHost, e.cfg.SMTPPort)
	auth := smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.SMTPHost)

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", e.cfg.From),
		fmt.Sprintf("To: %s", strings.Join(to, ",")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	d := &smtp.Client{}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: e.cfg.SMTPHost})
	if err != nil { return err }
	defer conn.Close()
	d, err = smtp.NewClient(conn, e.cfg.SMTPHost)
	if err != nil { return err }
	defer d.Quit()

	if err = d.Auth(auth); err != nil { return err }
	if err = d.Mail(e.cfg.From); err != nil { return err }
	for _, r := range to {
		if err = d.Rcpt(r); err != nil { return err }
	}
	w, err := d.Data()
	if err != nil { return err }
	if _, err = w.Write([]byte(msg)); err != nil { _ = w.Close(); return err }
	return w.Close()
}