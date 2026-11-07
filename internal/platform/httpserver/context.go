package httpserver

import (
	"context"
	"net/http"

	"mars-admin-go/internal/platform/requestctx"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	requestIDContextKey contextKey = "request-id"
	traceIDContextKey   contextKey = "trace-id"
)

// SetPathValue provides a small helper for tests while production routing
// receives values from Go 1.22+ net/http path patterns.
func SetPathValue(r *http.Request, key, value string) *http.Request {
	if r == nil {
		return nil
	}
	r.SetPathValue(key, value)
	return r
}

func ContextWithRequestMetadata(ctx context.Context, requestID string, traceID string) context.Context {
	ctx = context.WithValue(ctx, requestIDContextKey, requestID)
	ctx = context.WithValue(ctx, traceIDContextKey, traceID)
	ctx = requestctx.WithRequestID(ctx, requestID)
	ctx = requestctx.WithTraceID(ctx, traceID)
	return ctx
}

func requestMetadata(ctx context.Context) (string, string) {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	traceID, _ := ctx.Value(traceIDContextKey).(string)
	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		traceID = span.TraceID().String()
	}
	return requestID, traceID
}

func NewRequestID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
