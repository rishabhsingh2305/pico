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
	jwtSecretEnv          = "JWT_SECRET"
	publicBaseURLEnv      = "PUBLIC_BASE_URL"
	shortBaseURLEnv       = "SHORT_BASE_URL"
	bunnyAPIKeyEnv        = "BUNNY_API_KEY"
	corsAllowedOriginsEnv = "CORS_ALLOWED_ORIGINS"
	emailFromEnv          = "EMAIL_FROM"
	smtpAddrEnv           = "SMTP_ADDR"
	smtpUsernameEnv       = "SMTP_USERNAME"
	smtpPasswordEnv       = "SMTP_PASSWORD"
)

// Config contains authentication, email delivery, and short-link settings.
type Config struct {
	JWTSecret          string
	PublicBaseURL      string
	ShortBaseURL       string
	BunnyAPIKey        string
	CORSAllowedOrigins []string
	EmailFrom          string
	SMTPAddr           string
	SMTPUsername       string
	SMTPPassword       string
}

// ConfigFromEnv reads the authentication and email settings from the
// environment.
func ConfigFromEnv() (Config, error) {
	origins, err := parseCORSAllowedOrigins(os.Getenv(corsAllowedOriginsEnv))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		JWTSecret:          strings.TrimSpace(os.Getenv(jwtSecretEnv)),
		PublicBaseURL:      strings.TrimRight(strings.TrimSpace(os.Getenv(publicBaseURLEnv)), "/"),
		ShortBaseURL:       strings.TrimRight(strings.TrimSpace(os.Getenv(shortBaseURLEnv)), "/"),
		BunnyAPIKey:        strings.TrimSpace(os.Getenv(bunnyAPIKeyEnv)),
		CORSAllowedOrigins: origins,
		EmailFrom:          strings.TrimSpace(os.Getenv(emailFromEnv)),
		SMTPAddr:           strings.TrimSpace(os.Getenv(smtpAddrEnv)),
		SMTPUsername:       strings.TrimSpace(os.Getenv(smtpUsernameEnv)),
		SMTPPassword:       os.Getenv(smtpPasswordEnv),
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// AllowedOrigins returns the configured CORS origins, falling back to
// PUBLIC_BASE_URL when CORS_ALLOWED_ORIGINS is unset.
func (c Config) AllowedOrigins() []string {
	if len(c.CORSAllowedOrigins) > 0 {
		return c.CORSAllowedOrigins
	}
	return []string{c.PublicBaseURL}
}

func (c Config) validate() error {
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("%s must be at least 32 bytes", jwtSecretEnv)
	}

	if err := validateBaseURL(publicBaseURLEnv, c.PublicBaseURL); err != nil {
		return err
	}
	if c.ShortBaseURL != "" {
		if err := validateBaseURL(shortBaseURLEnv, c.ShortBaseURL); err != nil {
			return err
		}
	}
	if (c.ShortBaseURL == "") != (c.BunnyAPIKey == "") {
		return fmt.Errorf("%s and %s must be set together", shortBaseURLEnv, bunnyAPIKeyEnv)
	}

	for _, origin := range c.CORSAllowedOrigins {
		if origin == "*" {
			continue
		}
		if err := validateOrigin(origin); err != nil {
			return err
		}
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

func parseCORSAllowedOrigins(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	seen := make(map[string]struct{})
	origins := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		origin := strings.TrimRight(strings.TrimSpace(part), "/")
		if origin == "" {
			continue
		}
		if origin == "*" {
			return []string{"*"}, nil
		}
		if err := validateOrigin(origin); err != nil {
			return nil, err
		}

		key := strings.ToLower(origin)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		origins = append(origins, origin)
	}

	if len(origins) == 0 {
		return nil, fmt.Errorf("%s must contain at least one origin", corsAllowedOriginsEnv)
	}
	return origins, nil
}

func validateBaseURL(name, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute http or https URL", name)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must not contain a query or fragment", name)
	}
	if parsed.Path != "" {
		return fmt.Errorf("%s must not contain a path", name)
	}
	return nil
}

func validateOrigin(origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" ||
		parsed.Path != "" ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return fmt.Errorf("%s contains an invalid origin %q", corsAllowedOriginsEnv, origin)
	}
	return nil
}
