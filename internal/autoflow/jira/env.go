package jira

import (
	"errors"
	"fmt"
	"os"
)

// ErrMissingToken is returned when JIRA_API_TOKEN is not set. The token is
// never cached — only read from the env.
var ErrMissingToken = errors.New("JIRA_API_TOKEN env var is not set")

// Credentials are the three values required to make an authenticated Jira
// call. Host is bare (no scheme).
type Credentials struct {
	Host  string // e.g. "your-org.atlassian.net"
	Email string
	Token string
}

// ResolveCredentials returns the full set of values needed to talk to
// Jira. Precedence matches jira-env.sh: env vars win over the cache.
// Missing JIRA_API_TOKEN always fails; missing host/email fails only
// when neither the env var nor the cache supplies them.
func ResolveCredentials(root string) (*Credentials, error) {
	token := os.Getenv("JIRA_API_TOKEN")
	if token == "" {
		return nil, ErrMissingToken
	}

	host := HostFromSiteURL(os.Getenv("JIRA_SITE"))
	email := os.Getenv("JIRA_EMAIL")

	if host == "" || email == "" {
		c, err := Read(root)
		if err != nil && !errors.Is(err, ErrNoConfig) {
			return nil, err
		}
		if c != nil {
			if host == "" {
				host = HostFromSiteURL(c.SiteURL)
			}
			if email == "" {
				email = c.Email
			}
		}
	}

	var missing []string
	if host == "" {
		missing = append(missing, "JIRA_SITE (or cache .autoflow/jira-config.json siteUrl)")
	}
	if email == "" {
		missing = append(missing, "JIRA_EMAIL (or cache .autoflow/jira-config.json email)")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing jira credentials: %v", missing)
	}

	return &Credentials{Host: host, Email: email, Token: token}, nil
}
