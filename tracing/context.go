package tracing

import "context"

// spanContextKey is the key for storing spans in context.
type spanContextKey struct{}

// SpanFromContext retrieves the current span from the context.
func SpanFromContext(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	span, ok := ctx.Value(spanContextKey{}).(*Span)
	if !ok {
		return nil
	}
	return span
}

// ContextWithSpan returns a new context with the given span attached.
func ContextWithSpan(ctx context.Context, span *Span) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, spanContextKey{}, span)
}
