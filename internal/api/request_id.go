package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

var requestIDCounter uint64

func withRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	trimmed := strings.TrimSpace(requestID)
	if trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDContextKey{}, trimmed)
}

func requestIDFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if value, ok := r.Context().Value(requestIDContextKey{}).(string); ok {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return strings.TrimSpace(r.Header.Get(requestIDHeader))
}

func newRequestID(now time.Time) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err == nil {
		return fmt.Sprintf("req-%d-%s", now.UnixNano(), hex.EncodeToString(buf))
	}
	counter := atomic.AddUint64(&requestIDCounter, 1)
	return fmt.Sprintf("req-%d-%d", now.UnixNano(), counter)
}
