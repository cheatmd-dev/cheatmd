// Package httputil provides common HTTP client utilities.
package httputil

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// Get fetches the given URL with a GET request, returning the response body if
// the status code is 200 OK. The caller is responsible for closing the body.
func Get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return resp.Body, nil
}
