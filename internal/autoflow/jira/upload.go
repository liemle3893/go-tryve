package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/liemle3893/go-tryve/internal/autoflow/state"
)

// UploadResult is one line of the upload manifest printed to stdout.
type UploadResult struct {
	Filename string
	ID       string // jira attachment id (empty on failure)
	Error    error  // nil on success
}

// Upload uploads each file as an attachment to the given issue key. Returns
// one UploadResult per input file, in the same order. Partial success is
// possible: the slice always has len(files) entries.
func (c *Client) Upload(ctx context.Context, key string, files []string) ([]UploadResult, error) {
	if err := state.ValidateTicketKey(key); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("upload: at least one file required")
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			return nil, fmt.Errorf("upload: %w", err)
		}
	}

	results := make([]UploadResult, len(files))
	for i, path := range files {
		results[i] = UploadResult{Filename: filepath.Base(path)}
		id, err := c.uploadOne(ctx, key, path)
		if err != nil {
			results[i].Error = err
			continue
		}
		results[i].ID = id
	}
	return results, nil
}

// uploadOne posts one file to the /attachments endpoint. The Jira API
// requires the header X-Atlassian-Token: no-check to bypass XSRF checks
// for multipart uploads (see jira-upload.sh).
func (c *Client) uploadOne(ctx context.Context, key, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	urlPath := fmt.Sprintf("/rest/api/3/issue/%s/attachments", key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+urlPath, &body)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.creds.Email, c.creds.Token)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(raw, 200))
	}
	var arr []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) == 0 {
		return "?", nil // upload succeeded but couldn't parse id — caller keeps going
	}
	return arr[0].ID, nil
}
