package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// maxErrorBody is how much of an error body is read.
const maxErrorBody = 16 << 10

// mapStatus turns a non-2xx response into a typed error. It reads at
// most maxErrorBody, and never logs the body (it may echo the request).
func mapStatus(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrInvalidKey
	case http.StatusNotFound:
		return ErrModelNotFound
	case http.StatusTooManyRequests:
		return &RateLimitedError{RetryAfter: resp.Header.Get("Retry-After")}
	}
	return fmt.Errorf("%w (HTTP %d: %s)", ErrBadResponse, resp.StatusCode, shortBody(body))
}

// shortBody is the first line of an error body, trimmed, for messages.
func shortBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

// mapNetError turns a transport error into ErrUnreachable, unless the
// context ended first.
func mapNetError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var uerr *url.Error
	if errors.As(err, &uerr) {
		return fmt.Errorf("%w: %v", ErrUnreachable, uerr.Err)
	}
	return fmt.Errorf("%w: %v", ErrUnreachable, err)
}
