// Package httputil provides common HTTP client utilities.
package httputil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Get fetches the given URL with a GET request, returning the response body if
// the status code is 200 OK. The caller is responsible for closing the body.

func Get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if req.URL.Scheme != "https" {
		return nil, fmt.Errorf("insecure scheme %q: only https is allowed", req.URL.Scheme)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			if req.URL.Scheme != "https" {
				return errors.New("insecure redirect: only https is allowed")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return resp.Body, nil
}
