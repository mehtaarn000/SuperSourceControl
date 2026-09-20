// Package remote implements SSC's authenticated repository client.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"ssc/core"
	"strings"
	"time"
)

const MaxObjectBytes = 8 << 20
const maxMetadataBytes = 2 << 20

var ErrNotFound = errors.New("remote repository or object not found, or access denied")
var ErrConflict = errors.New("remote branch changed or diverged; fetch with pull and inspect history before retrying")

type Client struct {
	base  string
	token string
	http  *http.Client
}

// ValidateURL accepts the full /v1/repos/<name> endpoint. Credentials stay in SSC_TOKEN.
func ValidateURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
		return "", fmt.Errorf("expected a repository URL without credentials, query, or fragment")
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || ip == nil || !ip.IsLoopback() {
			return "", fmt.Errorf("use HTTPS, or HTTP with an explicit loopback IP")
		}
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	parts := strings.Split(u.Path, "/")
	if len(parts) != 4 || parts[1] != "v1" || parts[2] != "repos" || len(parts[3]) == 0 || len(parts[3]) > 100 {
		return "", fmt.Errorf("repository URL must end in /v1/repos/<name>")
	}
	for _, ch := range parts[3] {
		if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
			return "", fmt.Errorf("invalid repository name")
		}
	}
	return u.String(), nil
}

func NewClient(raw, token string) (*Client, error) {
	base, err := ValidateURL(raw)
	if err != nil {
		return nil, err
	}
	if token == "" || len(token) > 4096 || strings.ContainsAny(token, " \t\r\n") {
		return nil, fmt.Errorf("set SSC_TOKEN to your repository access token")
	}
	return &Client{base: base, token: token, http: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) request(ctx context.Context, method, path, kind string, body []byte) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("could not construct remote request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if method == http.MethodPut {
		if kind == "" {
			req.Header.Set("Content-Type", "application/json")
		} else {
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set("X-SSC-Object-Type", kind)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		// Do not echo server responses or URLs that could contain secrets.
		return nil, nil, fmt.Errorf("remote request failed; check connectivity, TLS trust, and server availability")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		switch resp.StatusCode {
		case 401:
			return nil, nil, fmt.Errorf("authentication failed; check SSC_TOKEN")
		case 403:
			return nil, nil, fmt.Errorf("remote token does not grant write access")
		case 404:
			return nil, nil, ErrNotFound
		case 409:
			return nil, nil, ErrConflict
		case 413:
			return nil, nil, fmt.Errorf("object exceeds server size limit")
		case 422:
			return nil, nil, fmt.Errorf("server rejected incomplete history or its graph limits were exceeded")
		default:
			return nil, nil, fmt.Errorf("remote returned HTTP %d", resp.StatusCode)
		}
	}
	limit := int64(maxMetadataBytes)
	if strings.HasPrefix(path, "/objects/") {
		limit = MaxObjectBytes
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, nil, fmt.Errorf("remote response was interrupted")
	}
	if int64(len(data)) > limit {
		return nil, nil, fmt.Errorf("remote response exceeds size limit")
	}
	return data, resp.Header, nil
}

func (c *Client) Refs(ctx context.Context) (map[string]string, error) {
	data, _, err := c.request(ctx, "GET", "/refs", "", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Refs map[string]string `json:"refs"`
	}
	if err := json.Unmarshal(data, &result); err != nil || result.Refs == nil {
		return nil, fmt.Errorf("invalid remote refs response")
	}
	if len(result.Refs) > 10000 {
		return nil, fmt.Errorf("too many remote branches")
	}
	for name, hash := range result.Refs {
		if !core.ValidBranchName(name) || len(name) > 255 || !core.ValidObjectID(hash) {
			return nil, fmt.Errorf("invalid remote branch reference")
		}
	}
	return result.Refs, nil
}
func (c *Client) Get(ctx context.Context, hash, kind string) ([]byte, error) {
	if !core.ValidObjectID(hash) {
		return nil, fmt.Errorf("invalid object ID")
	}
	data, header, err := c.request(ctx, "GET", "/objects/"+hash, "", nil)
	if err != nil {
		return nil, err
	}
	if header.Get("X-SSC-Object-Type") != kind {
		return nil, fmt.Errorf("remote object has the wrong type")
	}
	actual, _, err := core.InspectObject(kind, data)
	if err != nil || actual != hash {
		return nil, fmt.Errorf("remote object failed hash or format verification")
	}
	return data, nil
}
func (c *Client) Has(ctx context.Context, hash, kind string) (bool, error) {
	if !core.ValidObjectID(hash) {
		return false, fmt.Errorf("invalid object ID")
	}
	_, header, err := c.request(ctx, "HEAD", "/objects/"+hash, "", nil)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if header.Get("X-SSC-Object-Type") != kind {
		return false, fmt.Errorf("remote object has the wrong type")
	}
	return true, nil
}
func (c *Client) Put(ctx context.Context, hash, kind string, data []byte) error {
	actual, _, err := core.InspectObject(kind, data)
	if err != nil || actual != hash || len(data) > MaxObjectBytes {
		return fmt.Errorf("invalid or oversized local object")
	}
	_, _, err = c.request(ctx, "PUT", "/objects/"+hash, kind, data)
	return err
}
func (c *Client) Update(ctx context.Context, branch, old, next string) error {
	if !core.ValidBranchName(branch) || !core.ValidObjectID(next) || (old != "" && !core.ValidObjectID(old)) {
		return fmt.Errorf("invalid branch update")
	}
	data, _ := json.Marshal(map[string]string{"old": old, "new": next})
	_, _, err := c.request(ctx, "PUT", "/refs/"+url.PathEscape(branch), "", data)
	return err
}
