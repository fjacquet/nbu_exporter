package exporter

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerWrapper provides nil-safe tracing by using noop.TracerProvider as default.
// All span methods are guaranteed safe to call - no nil-checks needed.
type TracerWrapper struct {
	tracer trace.Tracer
}

// NewTracerWrapper creates a TracerWrapper from a TracerProvider.
// If tp is nil, uses noop.NewTracerProvider() to ensure all operations are safe.
func NewTracerWrapper(tp trace.TracerProvider, instrumentationName string) *TracerWrapper {
	if tp == nil {
		tp = noop.NewTracerProvider()
	}
	return &TracerWrapper{
		tracer: tp.Tracer(instrumentationName),
	}
}

// StartSpan creates a new span with the given operation name and kind.
// Always returns a valid span (noop if tracing disabled) - no nil-check needed.
func (w *TracerWrapper) StartSpan(ctx context.Context, operation string, kind trace.SpanKind) (context.Context, trace.Span) {
	return w.tracer.Start(ctx, operation, trace.WithSpanKind(kind))
}

// Tracer returns the underlying tracer for advanced use cases.
func (w *TracerWrapper) Tracer() trace.Tracer {
	return w.tracer
}

// Deprecated: Use TracerWrapper.StartSpan instead. This function will be removed
// after all call sites are migrated to use TracerWrapper.
//
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
