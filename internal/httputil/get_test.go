package httputil

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setupInsecureTransport modifies the global DefaultTransport to allow insecure TLS
// for the duration of the test, restoring it afterwards.
func setupInsecureTransport(t *testing.T) {
	t.Helper()
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Skip("DefaultTransport is not *http.Transport, cannot skip TLS verify")
	}

	oldConfig := defaultTransport.TLSClientConfig
	if defaultTransport.TLSClientConfig == nil {
		defaultTransport.TLSClientConfig = &tls.Config{}
	} else {
		// Clone it just to be safe if it existed
		defaultTransport.TLSClientConfig = defaultTransport.TLSClientConfig.Clone()
	}
	defaultTransport.TLSClientConfig.InsecureSkipVerify = true

	t.Cleanup(func() {
		defaultTransport.TLSClientConfig = oldConfig
	})
}

func TestGet_HTTPS_Enforcement(t *testing.T) {
	ctx := context.Background()

	// 1. Direct HTTP request should fail early
	_, err := Get(ctx, "http://example.com")
	if err == nil || !strings.Contains(err.Error(), "only https is allowed") {
		t.Fatalf("expected insecure scheme error, got: %v", err)
	}
}

func TestGet_Success(t *testing.T) {
	setupInsecureTransport(t)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer ts.Close()

	ctx := context.Background()
	body, err := Get(ctx, ts.URL)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	defer body.Close()

	b, _ := io.ReadAll(body)
	if string(b) != "success" {
		t.Errorf("expected body 'success', got: %s", string(b))
	}
}

func TestGet_Non200Status(t *testing.T) {
	setupInsecureTransport(t)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	ctx := context.Background()
	_, err := Get(ctx, ts.URL)
	if err == nil || !strings.Contains(err.Error(), "unexpected status 404") {
		t.Fatalf("expected 404 error, got: %v", err)
	}
}

func TestGet_RedirectToHTTP(t *testing.T) {
	setupInsecureTransport(t)

	// Insecure server we want to redirect to
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer httpServer.Close()

	// Secure server that redirects
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, httpServer.URL, http.StatusFound)
	}))
	defer ts.Close()

	ctx := context.Background()
	_, err := Get(ctx, ts.URL)
	if err == nil || !strings.Contains(err.Error(), "insecure redirect: only https is allowed") {
		t.Fatalf("expected insecure redirect error, got: %v", err)
	}
}

func TestGet_RedirectLoop(t *testing.T) {
	setupInsecureTransport(t)

	var ts *httptest.Server
	ts = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, ts.URL, http.StatusFound)
	}))
	defer ts.Close()

	ctx := context.Background()
	_, err := Get(ctx, ts.URL)
	if err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Fatalf("expected redirect loop error, got: %v", err)
	}
}
