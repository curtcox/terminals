package repl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (s *state) fetchJSON(ctx context.Context, route string) (map[string]any, error) {
	return s.doJSON(ctx, http.MethodGet, route, "", nil)
}

func (s *state) fetchJSONQuery(ctx context.Context, route string, query url.Values) (map[string]any, error) {
	base, err := url.JoinPath(s.adminURL, route)
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	parsed.RawQuery = query.Encode()
	return s.doJSON(ctx, http.MethodGet, parsed.String(), "", nil)
}

func (s *state) deleteJSON(ctx context.Context, route string) (map[string]any, error) {
	return s.doJSON(ctx, http.MethodDelete, route, "", nil)
}

func (s *state) fetchTextQuery(ctx context.Context, route string, query url.Values) (string, error) {
	base, err := url.JoinPath(s.adminURL, route)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("admin request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (s *state) postFormJSON(ctx context.Context, route string, form url.Values) (map[string]any, error) {
	if form == nil {
		form = url.Values{}
	}
	return s.doJSON(ctx, http.MethodPost, route, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
}

func (s *state) postJSON(ctx context.Context, route string, payload map[string]any) (map[string]any, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return s.doJSON(ctx, http.MethodPost, route, "application/json", bytes.NewReader(b))
}

func (s *state) doJSON(ctx context.Context, method, route, contentType string, body io.Reader) (map[string]any, error) {
	u := strings.TrimSpace(route)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		var err error
		u, err = url.JoinPath(s.adminURL, route)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("admin request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}
