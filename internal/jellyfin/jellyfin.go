// Package jellyfin asks a Jellyfin server to scan its libraries, so new
// downloads show up without waiting for its scheduled scan.
package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// requestTimeout bounds each request to Jellyfin.
const requestTimeout = 10 * time.Second

// Client talks to one Jellyfin server.
type Client struct {
	// URL is the server's base URL, e.g. http://jellyfin:8096.
	URL string
	// APIKey is created in Jellyfin's dashboard under API Keys.
	APIKey string
	// HTTP is the client to use; nil uses one with requestTimeout.
	HTTP *http.Client
}

// ErrUnauthorized means Jellyfin rejected the API key.
//
// Errors here start with "Jellyfin", a name, and are shown to users as is.
var ErrUnauthorized = errors.New("Jellyfin rejected the API key") //nolint:staticcheck // ST1005, see above

// Refresh starts a scan of all libraries. Jellyfin runs it in the
// background; Refresh returns once it has been accepted.
func (c Client) Refresh(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodPost, "/Library/Refresh")
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Check verifies the URL and API key, returning a description of the server
// such as "Home (Jellyfin 10.11.2)".
func (c Client) Check(ctx context.Context) (string, error) {
	resp, err := c.do(ctx, http.MethodGet, "/System/Info")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var info struct {
		ServerName string `json:"ServerName"`
		Version    string `json:"Version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info); err != nil {
		return "", fmt.Errorf("%s doesn't look like a Jellyfin server", c.URL)
	}
	return fmt.Sprintf("%s (Jellyfin %s)", info.ServerName, info.Version), nil
}

func (c Client) do(ctx context.Context, method, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.URL, "/")+path, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid Jellyfin URL: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("MediaBrowser Token=%q", c.APIKey))

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: requestTimeout}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't reach Jellyfin at %s: %w", c.URL, err)
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		_ = resp.Body.Close()
		return nil, ErrUnauthorized
	case resp.StatusCode >= 300:
		_ = resp.Body.Close()
		return nil, fmt.Errorf("Jellyfin answered %s", resp.Status) //nolint:staticcheck // ST1005, see ErrUnauthorized
	}
	return resp, nil
}

// Notifier refreshes Jellyfin's libraries when downloads finish, using the
// URL and key current at that moment. It does nothing until both are set.
// It implements media.LibraryNotifier.
type Notifier struct {
	// Config returns the current URL and API key.
	Config func() (url, apiKey string)
	HTTP   *http.Client
}

// LibraryChanged asks Jellyfin to scan, if configured.
func (n Notifier) LibraryChanged(ctx context.Context) error {
	url, key := n.Config()
	if url == "" || key == "" {
		return nil
	}
	return Client{URL: url, APIKey: key, HTTP: n.HTTP}.Refresh(ctx)
}
