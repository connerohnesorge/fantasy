package tracing

import "context"

// Context keys for trace and span propagation.
type contextKey int

const (
	traceKey contextKey = iota
	spanKey
)

// withTrace returns a new context with the trace embedded.
func withTrace(ctx context.Context, trace *Trace) context.Context {
	return context.WithValue(ctx, traceKey, trace)
}

// TraceFromContext retrieves the current trace from the context.
func TraceFromContext(ctx context.Context) *Trace {
	if trace, ok := ctx.Value(traceKey).(*Trace); ok {
		return trace
	}
	return nil
}

// withSpan returns a new context with the span embedded.
func withSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanKey, span)
}

// SpanFromContext retrieves the current span from the context.
func SpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanKey).(*Span); ok {
		return span
	}
	return nil
}

// WithParentSpan returns a context that will make the given span
// the parent of any new spans created.
func WithParentSpan(ctx context.Context, span *Span) context.Context {
	return withSpan(ctx, span)
}
