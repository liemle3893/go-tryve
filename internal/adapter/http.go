package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// HTTPAdapter executes HTTP requests against a target base URL.
// It maintains a persistent http.Client with cookie jar support across requests.
type HTTPAdapter struct {
	baseURL string
	client  *http.Client
}

// NewHTTPAdapter constructs an HTTPAdapter targeting the given baseURL.
// Connect must be called before Execute or Health.
func NewHTTPAdapter(baseURL string) *HTTPAdapter {
	return &HTTPAdapter{baseURL: strings.TrimRight(baseURL, "/")}
}

// Name returns the adapter's registered identifier.
func (a *HTTPAdapter) Name() string { return "http" }

// Connect initialises the http.Client with a 30-second timeout and a cookie jar
// so cookies are persisted across requests within the same adapter instance.
func (a *HTTPAdapter) Connect(_ context.Context) error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return tryve.ConnectionError("http", "failed to create cookie jar", err)
	}
	a.client = &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}
	return nil
}

// Close releases idle connections held by the HTTP client.
func (a *HTTPAdapter) Close(_ context.Context) error {
	if a.client != nil {
		a.client.CloseIdleConnections()
	}
	return nil
}

// Health performs a lightweight HEAD request to baseURL to verify connectivity.
func (a *HTTPAdapter) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, a.baseURL, nil)
	if err != nil {
		return tryve.ConnectionError("http", "health check: failed to build request", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return tryve.ConnectionError("http", "health check: request failed", err)
	}
	_ = resp.Body.Close()
	return nil
}

// Execute dispatches the named action with the given parameters.
// Only the "request" action is supported; any other name returns an ADAPTER_ERROR.
func (a *HTTPAdapter) Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error) {
	if action != "request" {
		return nil, tryve.AdapterError("http", action,
			fmt.Sprintf("unsupported action %q: only \"request\" is supported", action), nil)
	}
	return a.executeRequest(ctx, params)
}

// executeRequest builds and sends an HTTP request from the provided params map,
// then parses the response into a StepResult.
func (a *HTTPAdapter) executeRequest(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	method := stringParam(params, "method", "GET")
	rawURL := stringParam(params, "url", "")
	if rawURL == "" {
		return nil, tryve.AdapterError("http", "request", "missing required param: url", nil)
	}

	// Resolve URL: absolute URLs are used directly; relative paths are prefixed
	// with baseURL.
	targetURL := rawURL
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		targetURL = a.baseURL + "/" + strings.TrimLeft(rawURL, "/")
	}

	// Append query parameters when provided.
	if q, ok := params["query"]; ok {
		if qMap, ok := q.(map[string]any); ok && len(qMap) > 0 {
			parsed, err := url.Parse(targetURL)
			if err != nil {
				return nil, tryve.AdapterError("http", "request", "invalid url", err)
			}
			qs := parsed.Query()
			for k, v := range qMap {
				qs.Set(k, fmt.Sprintf("%v", v))
			}
			parsed.RawQuery = qs.Encode()
			targetURL = parsed.String()
		}
	}

	// Build request body for methods that carry a payload.
	var bodyReader io.Reader
	hasBody := false
	if bodyVal, ok := params["body"]; ok && bodyVal != nil {
		upperMethod := strings.ToUpper(method)
		if upperMethod != http.MethodGet && upperMethod != http.MethodHead {
			encoded, err := json.Marshal(bodyVal)
			if err != nil {
				return nil, tryve.AdapterError("http", "request", "failed to marshal body", err)
			}
			bodyReader = bytes.NewReader(encoded)
			hasBody = true
		}
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), targetURL, bodyReader)
	if err != nil {
		return nil, tryve.AdapterError("http", "request", "failed to build request", err)
	}

	// Auto-set Content-Type when a body is present and the caller has not
	// already specified a Content-Type header.
	if hasBody {
		if h, ok := params["headers"]; ok {
			if hMap, ok := h.(map[string]any); ok {
				if _, has := hMap["Content-Type"]; !has {
					req.Header.Set("Content-Type", "application/json")
				}
			} else {
				req.Header.Set("Content-Type", "application/json")
			}
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Apply caller-supplied headers.
	if h, ok := params["headers"]; ok {
		if hMap, ok := h.(map[string]any); ok {
			for k, v := range hMap {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}
	}

	var resp *http.Response
	duration, err := MeasureDuration(func() error {
		var doErr error
		resp, doErr = a.client.Do(req)
		return doErr
	})
	if err != nil {
		return nil, tryve.AdapterError("http", "request", "request failed", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, tryve.AdapterError("http", "request", "failed to read response body", err)
	}

	// Parse body: JSON when Content-Type indicates it, plain string otherwise.
	var parsedBody any = string(rawBody)
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") && len(rawBody) > 0 {
		var decoded any
		if jsonErr := json.Unmarshal(rawBody, &decoded); jsonErr == nil {
			parsedBody = decoded
		}
		// On JSON parse error, parsedBody retains the raw string value.
	}

	// Collect response headers as a flat key→single-value map.
	respHeaders := make(map[string]any, len(resp.Header))
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	data := map[string]any{
		"status":     float64(resp.StatusCode),
		"statusText": resp.Status,
		"headers":    respHeaders,
		"body":       parsedBody,
		"duration":   float64(duration.Milliseconds()),
	}

	meta := map[string]any{
		"method": method,
		"url":    targetURL,
	}

	return SuccessResult(data, duration, meta), nil
}

// stringParam retrieves a string value from params by key, returning def when
// the key is absent or the value cannot be asserted to string.
func stringParam(params map[string]any, key, def string) string {
	v, ok := params[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}
