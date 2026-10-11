package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	To      string
	Subject string
	Body    string
	Ref     string
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type SMTPSender struct {
	cfg  Config
	from *mail.Address
}

func NewSMTPSender(cfg Config) (*SMTPSender, error) {
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("parse sender address: %w", err)
	}
	return &SMTPSender{cfg: cfg, from: from}, nil
}

func (s *SMTPSender) Send(ctx context.Context, m Message) error {
	raw, err := build(s.from, m, time.Now())
	if err != nil {
		return err
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port)))
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	c, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if s.cfg.User != "" {
		if err := c.Auth(smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(s.from.Address); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(m.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("finish message: %w", err)
	}
	return c.Quit()
}

func build(from *mail.Address, m Message, now time.Time) ([]byte, error) {
	if strings.ContainsAny(m.To, "\r\n") {
		return nil, fmt.Errorf("invalid recipient")
	}

	id := make([]byte, 12)
	rand.Read(id)
	domain := from.Address[strings.LastIndex(from.Address, "@")+1:]

	var b bytes.Buffer
	header := func(k, v string) {
		fmt.Fprintf(&b, "%s: %s\r\n", k, v)
	}
	header("From", from.String())
	header("To", m.To)
	header("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	header("Date", now.Format(time.RFC1123Z))
	header("Message-ID", fmt.Sprintf("<%s@%s>", hex.EncodeToString(id), domain))
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=UTF-8")
	header("Content-Transfer-Encoding", "quoted-printable")
	b.WriteString("\r\n")

	qp := quotedprintable.NewWriter(&b)
	body := strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n")
	if _, err := qp.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
