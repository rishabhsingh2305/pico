// Package email sends transactional email over SMTP.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

// Config contains the SMTP settings.
type Config struct {
	Addr     string
	Username string
	Password string
	From     string
	Timeout  time.Duration
}

// Sender sends transactional email over SMTP.
type Sender struct {
	addr        string
	host        string
	username    string
	password    string
	from        string
	fromAddress string
	timeout     time.Duration
}

// NewSMTP returns an SMTP sender for cfg.
func NewSMTP(cfg Config) (*Sender, error) {
	addr := strings.TrimSpace(cfg.Addr)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid smtp address: %w", err)
	}
	if host == "" {
		return nil, fmt.Errorf("invalid smtp address: empty host")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("invalid smtp address: bad port %q", port)
	}

	from := strings.TrimSpace(cfg.From)
	if strings.ContainsAny(from, "\r\n") {
		return nil, fmt.Errorf("invalid email from address")
	}
	parsed, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid email from address: %w", err)
	}

	if (cfg.Username == "") != (cfg.Password == "") {
		return nil, fmt.Errorf("smtp username and password must be set together")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &Sender{
		addr:        addr,
		host:        host,
		username:    cfg.Username,
		password:    cfg.Password,
		from:        from,
		fromAddress: parsed.Address,
		timeout:     timeout,
	}, nil
}

// SendVerification sends a verification link and code to the recipient.
func (s *Sender) SendVerification(ctx context.Context, to, name, verifyURL, code string) error {
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("invalid recipient address")
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}

	message := BuildVerificationMessage(s.from, to, name, verifyURL, code)

	dialer := net.Dialer{Timeout: s.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer func() { _ = conn.Close() }()

	deadline := time.Now().Add(s.timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set smtp deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer func() { _ = client.Close() }()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		}); err != nil {
			return fmt.Errorf("start smtp tls: %w", err)
		}
	}
	if s.username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.fromAddress); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}
