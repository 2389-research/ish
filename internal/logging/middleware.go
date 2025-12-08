// ABOUTME: HTTP request logging middleware.
// ABOUTME: Captures method, path, status, duration, request/response bodies, and stores in database.

package logging

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

const maxBodySize = 10 * 1024 // 10KB limit for body capture

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
	body       *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	// Capture response body (up to maxBodySize)
	if rw.body.Len() < maxBodySize {
		toCopy := len(b)
		if rw.body.Len()+toCopy > maxBodySize {
			toCopy = maxBodySize - rw.body.Len()
		}
		rw.body.Write(b[:toCopy])
	}
	return rw.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker to support WebSocket upgrades
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

// Middleware logs all HTTP requests to the database
func Middleware(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for health checks and admin UI assets
			if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// Determine plugin
			pluginName := GetPluginFromPath(r.URL.Path)

			// Capture request body (if present)
			var requestBody string
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
				if err == nil {
					requestBody = string(bodyBytes)
					// Restore the body for the handler to read
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			start := time.Now()
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     200,
				body:           &bytes.Buffer{},
			}

			// Call the next handler
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Milliseconds()

			// Get user from context (if authenticated)
			userID := auth.UserFromContext(r.Context())

			// Get client IP
			ip := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ip = strings.Split(forwarded, ",")[0]
			}

			// Log to database (fire and forget)
			go s.LogRequest(&store.RequestLog{
				PluginName:   pluginName,
				Method:       r.Method,
				Path:         r.URL.Path,
				StatusCode:   wrapped.statusCode,
				DurationMs:   int(duration),
				UserID:       userID,
				IPAddress:    ip,
				UserAgent:    r.Header.Get("User-Agent"),
				RequestBody:  requestBody,
				ResponseBody: wrapped.body.String(),
			})
		})
	}
}
