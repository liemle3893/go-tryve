package jira

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DownloadResult is one line of the download manifest.
type DownloadResult struct {
	Attachment Attachment
	Path       string // local destination when written
	Error      error
}

// Download fetches every attachment on the given issue into destDir. The
// directory is created if it does not exist. Attachments keep their Jira
// filename. Partial success: returns one DownloadResult per attachment,
// even for those that failed.
func (c *Client) Download(ctx context.Context, key, destDir string) ([]DownloadResult, error) {
	atts, err := c.ListAttachments(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(atts) == 0 {
		return nil, nil
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", destDir, err)
	}

	out := make([]DownloadResult, len(atts))
	for i, att := range atts {
		out[i] = DownloadResult{Attachment: att}
		dest := filepath.Join(destDir, att.Filename)
		if err := c.downloadTo(ctx, att.Content, dest); err != nil {
			out[i].Error = err
			continue
		}
		out[i].Path = dest
	}
	return out, nil
}

func (c *Client) downloadTo(ctx context.Context, contentURL, dest string) error {
	rc, err := c.GetAttachment(ctx, contentURL)
	if err != nil {
		return err
	}
	defer rc.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, rc); err != nil {
		return err
	}
	return nil
}
