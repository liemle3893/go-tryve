package jira

import (
	"errors"
	"testing"
)

func TestResolveCredentials_FromCache(t *testing.T) {
	root := t.TempDir()
	_, _ = Set(root, "c", "https://x.atlassian.net", "P", "me@x")
	t.Setenv("JIRA_API_TOKEN", "tok")
	t.Setenv("JIRA_SITE", "")
	t.Setenv("JIRA_EMAIL", "")

	creds, err := ResolveCredentials(root)
	if err != nil {
		t.Fatal(err)
	}
	if creds.Host != "x.atlassian.net" {
		t.Errorf("host: got %q", creds.Host)
	}
	if creds.Email != "me@x" {
		t.Errorf("email: got %q", creds.Email)
	}
	if creds.Token != "tok" {
		t.Errorf("token: got %q", creds.Token)
	}
}

func TestResolveCredentials_EnvWinsOverCache(t *testing.T) {
	root := t.TempDir()
	_, _ = Set(root, "c", "https://cache.atlassian.net", "P", "cache@x")
	t.Setenv("JIRA_API_TOKEN", "tok")
	t.Setenv("JIRA_SITE", "env.atlassian.net")
	t.Setenv("JIRA_EMAIL", "env@x")

	creds, err := ResolveCredentials(root)
	if err != nil {
		t.Fatal(err)
	}
	if creds.Host != "env.atlassian.net" || creds.Email != "env@x" {
		t.Errorf("env should override cache: got host=%q email=%q", creds.Host, creds.Email)
	}
}

func TestResolveCredentials_MissingToken(t *testing.T) {
	t.Setenv("JIRA_API_TOKEN", "")
	_, err := ResolveCredentials(t.TempDir())
	if !errors.Is(err, ErrMissingToken) {
		t.Errorf("want ErrMissingToken, got %v", err)
	}
}

func TestResolveCredentials_NoHostNoEmail(t *testing.T) {
	t.Setenv("JIRA_API_TOKEN", "tok")
	t.Setenv("JIRA_SITE", "")
	t.Setenv("JIRA_EMAIL", "")

	_, err := ResolveCredentials(t.TempDir())
	if err == nil {
		t.Errorf("expected error when host+email unresolvable")
	}
}
