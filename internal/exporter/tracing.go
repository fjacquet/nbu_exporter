package exporter

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// createSpan creates a new span with the given operation name and span kind.
// Returns the original context and nil span if tracer is nil (tracing disabled).
//
// Parameters:
//   - ctx: The parent context for the span
//   - tracer: The OpenTelemetry tracer (may be nil if tracing is disabled)
//   - operation: The name of the operation being traced
//   - kind: The span kind (e.g., SpanKindClient, SpanKindInternal)
//
// Returns:
//   - context.Context: A new context containing the span (or original context if tracer is nil)
//   - trace.Span: The created span (or nil if tracer is nil)
//
// Example:
//
//	ctx, span := createSpan(ctx, tracer, "FetchStorage", trace.SpanKindClient)
//	if span != nil {
//	    defer span.End()
//	}
func createSpan(ctx context.Context, tracer trace.Tracer, operation string, kind trace.SpanKind) (context.Context, trace.Span) {
	if tracer == nil {
		return ctx, nil
	}
	return tracer.Start(ctx, operation, trace.WithSpanKind(kind))
}
