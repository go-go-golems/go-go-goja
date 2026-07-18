package fetch

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultTimeout          = 30 * time.Second
	DefaultMaxResponseBytes = 4 << 20
)

type Policy struct {
	AllowedOrigins   []string
	Timeout          time.Duration
	MaxResponseBytes int64
	Credentials      CredentialPolicy
}

type CredentialPolicy struct {
	AllowEnv     bool
	AllowFiles   bool
	AllowedFiles []string
}

func (p Policy) normalized() Policy {
	if p.Timeout <= 0 {
		p.Timeout = DefaultTimeout
	}
	if p.MaxResponseBytes <= 0 {
		p.MaxResponseBytes = DefaultMaxResponseBytes
	}
	return p
}

func (p Policy) CheckURL(raw string) (*url.URL, error) {
	p = p.normalized()
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("fetch only supports http and https URLs")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("fetch url requires a host")
	}
	if len(p.AllowedOrigins) == 0 {
		return u, nil
	}
	for _, pattern := range p.AllowedOrigins {
		if originPatternMatches(strings.TrimSpace(pattern), u) {
			return u, nil
		}
	}
	return nil, fmt.Errorf("fetch target origin %s is not allowed", originOf(u))
}

func (p Policy) CheckCredentialFile(path string) error {
	p = p.normalized()
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("credential file path is required")
	}
	if !p.Credentials.AllowFiles {
		return fmt.Errorf("credential file sources are not allowed by fetch module policy")
	}
	if len(p.Credentials.AllowedFiles) == 0 {
		return nil
	}
	clean, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for _, allowed := range p.Credentials.AllowedFiles {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			return err
		}
		if clean == allowedAbs {
			return nil
		}
	}
	return fmt.Errorf("credential file %q is not allowed by fetch module policy", path)
}

func (p Policy) CheckCredentialEnv(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("credential env var name is required")
	}
	if !p.normalized().Credentials.AllowEnv {
		return fmt.Errorf("credential env sources are not allowed by fetch module policy")
	}
	return nil
}

func originPatternMatches(pattern string, u *url.URL) bool {
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	origin := originOf(u)
	if pattern == origin {
		return true
	}
	if strings.HasSuffix(pattern, ":*") {
		base := strings.TrimSuffix(pattern, ":*")
		return u.Scheme+"://"+u.Hostname() == base
	}
	return false
}

func originOf(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
