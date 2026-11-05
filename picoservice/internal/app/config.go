package app

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	jwtSecretEnv     = "JWT_SECRET"
	publicBaseURLEnv = "PUBLIC_BASE_URL"
	emailFromEnv     = "EMAIL_FROM"
	smtpAddrEnv      = "SMTP_ADDR"
	smtpUsernameEnv  = "SMTP_USERNAME"
	smtpPasswordEnv  = "SMTP_PASSWORD"
)

// Config contains authentication and email delivery settings.
type Config struct {
	JWTSecret     string
	PublicBaseURL string
	EmailFrom     string
	SMTPAddr      string
	SMTPUsername  string
	SMTPPassword  string
}

// ConfigFromEnv reads the authentication and email settings from the
// environment.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		JWTSecret:     strings.TrimSpace(os.Getenv(jwtSecretEnv)),
		PublicBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv(publicBaseURLEnv)), "/"),
		EmailFrom:     strings.TrimSpace(os.Getenv(emailFromEnv)),
		SMTPAddr:      strings.TrimSpace(os.Getenv(smtpAddrEnv)),
		SMTPUsername:  strings.TrimSpace(os.Getenv(smtpUsernameEnv)),
		SMTPPassword:  os.Getenv(smtpPasswordEnv),
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("%s must be at least 32 bytes", jwtSecretEnv)
	}

	baseURL, err := url.Parse(c.PublicBaseURL)
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" {
		return fmt.Errorf("%s must be an absolute http or https URL", publicBaseURLEnv)
	}
	if baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return fmt.Errorf("%s must not contain a query or fragment", publicBaseURLEnv)
	}

	if c.EmailFrom == "" {
		return fmt.Errorf("%s is not set", emailFromEnv)
	}
	if _, err := mail.ParseAddress(c.EmailFrom); err != nil {
		return fmt.Errorf("%s must be a valid email address: %w", emailFromEnv, err)
	}

	if c.SMTPAddr == "" {
		return fmt.Errorf("%s is not set", smtpAddrEnv)
	}
	host, port, err := net.SplitHostPort(c.SMTPAddr)
	if err != nil || host == "" {
		return fmt.Errorf("%s must be a host:port address", smtpAddrEnv)
	}
	if portNumber, err := strconv.Atoi(port); err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("%s has an invalid port", smtpAddrEnv)
	}

	if (c.SMTPUsername == "") != (c.SMTPPassword == "") {
		return fmt.Errorf("%s and %s must be set together", smtpUsernameEnv, smtpPasswordEnv)
	}
	return nil
}
