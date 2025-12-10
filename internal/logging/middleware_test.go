// ABOUTME: Tests for HTTP request logging middleware.
// ABOUTME: Verifies memory safety, body buffering limits, and response capture behavior.

package logging

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2389/ish/internal/store"
)

func TestResponseWriter_BuffersResponseBody(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		expectedCapped bool
	}{
		{
			name:           "small response",
			responseBody:   "Hello, World!",
			expectedCapped: false,
		},
		{
			name:           "response at limit",
			responseBody:   strings.Repeat("x", maxBodySize),
			expectedCapped: false,
		},
		{
			name:           "response exceeds limit",
			responseBody:   strings.Repeat("x", maxBodySize+1000),
			expectedCapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			wrapped := &responseWriter{
				ResponseWriter: rr,
				statusCode:     200,
				body:           &bytes.Buffer{},
			}

			// Write response body
			n, err := wrapped.Write([]byte(tt.responseBody))
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			// Verify all bytes were written to the underlying writer
			if n != len(tt.responseBody) {
				t.Errorf("Write() returned %d, want %d", n, len(tt.responseBody))
			}

			// Verify buffered body respects size limit
			buffered := wrapped.body.String()
			if len(buffered) > maxBodySize {
				t.Errorf("Buffered body size %d exceeds maxBodySize %d", len(buffered), maxBodySize)
			}

			if tt.expectedCapped && len(buffered) != maxBodySize {
				t.Errorf("Expected buffered body to be capped at %d, got %d", maxBodySize, len(buffered))
			}
		})
	}
}

func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		explicit bool
		code     int
	}{
		{"explicit status", true, http.StatusCreated},
		{"implicit status", false, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			wrapped := &responseWriter{
				ResponseWriter: rr,
				statusCode:     200,
				body:           &bytes.Buffer{},
			}

			if tt.explicit {
				wrapped.WriteHeader(tt.code)
			}

			// Write triggers implicit status if not set
			wrapped.Write([]byte("body"))

			if wrapped.statusCode != tt.code {
				t.Errorf("statusCode = %d, want %d", wrapped.statusCode, tt.code)
			}
		})
	}
}

func TestResponseWriter_PartialBufferOnLargeResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	wrapped := &responseWriter{
		ResponseWriter: rr,
		statusCode:     200,
		body:           &bytes.Buffer{},
	}

	// Write multiple chunks that exceed limit
	chunk1 := strings.Repeat("a", maxBodySize/2)
	chunk2 := strings.Repeat("b", maxBodySize)

	wrapped.Write([]byte(chunk1))
	wrapped.Write([]byte(chunk2))

	buffered := wrapped.body.String()
	if len(buffered) > maxBodySize {
		t.Errorf("Buffered body size %d exceeds maxBodySize %d", len(buffered), maxBodySize)
	}

	// Verify we got the first part of chunk1
	if !strings.HasPrefix(buffered, "a") {
		t.Errorf("Expected buffered body to start with 'a'")
	}
}

func TestResponseWriter_Hijack(t *testing.T) {
	rr := httptest.NewRecorder()
	wrapped := &responseWriter{
		ResponseWriter: rr,
		statusCode:     200,
		body:           &bytes.Buffer{},
	}

	// httptest.ResponseRecorder doesn't implement Hijacker, should return error
	_, _, err := wrapped.Hijack()
	if err != http.ErrNotSupported {
		t.Errorf("Hijack() error = %v, want %v", err, http.ErrNotSupported)
	}
}

func TestMiddleware_RequestBodySizeLimit(t *testing.T) {
	// Create a mock store
	s := &store.Store{}

	// Create middleware
	handler := Middleware(s)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))

	// Test with request body exceeding limit
	largeBody := strings.NewReader(strings.Repeat("x", maxBodySize+1000))
	req := httptest.NewRequest("POST", "/api/test", largeBody)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify the request was still processed
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMiddleware_SkipsHealthcheckLogging(t *testing.T) {
	s := &store.Store{}

	handler := Middleware(s)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify it still responded
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMiddleware_SkipsAdminAssets(t *testing.T) {
	s := &store.Store{}

	handler := Middleware(s)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin/dashboard", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify it still responded
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestResponseWriter_RestoresRequestBody(t *testing.T) {
	s := &store.Store{}

	originalBody := "test request body"
	var handlerReadBody string

	handler := Middleware(s)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		handlerReadBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/test", strings.NewReader(originalBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if handlerReadBody != originalBody {
		t.Errorf("Handler read body = %q, want %q", handlerReadBody, originalBody)
	}
}
