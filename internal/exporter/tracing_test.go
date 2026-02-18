package exporter

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewTracerWrapper_NilProvider(t *testing.T) {
	// When TracerProvider is nil, should use noop
	wrapper := NewTracerWrapper(nil, "test-component")

	if wrapper == nil {
		t.Fatal("expected non-nil wrapper")
	}
	if wrapper.tracer == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestNewTracerWrapper_WithProvider(t *testing.T) {
	// When TracerProvider is provided, should use it
	tp := noop.NewTracerProvider()
	wrapper := NewTracerWrapper(tp, "test-component")

	if wrapper == nil {
		t.Fatal("expected non-nil wrapper")
	}
	if wrapper.tracer == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestTracerWrapper_StartSpan_NilSafe(t *testing.T) {
	// StartSpan should always return valid span, even with nil provider
	wrapper := NewTracerWrapper(nil, "test-component")
	ctx := context.Background()

	_, span := wrapper.StartSpan(ctx, "test-operation", trace.SpanKindClient)

	// Span should never be nil
	if span == nil {
		t.Fatal("expected non-nil span")
	}

	// All span methods should be safe to call
	span.SetAttributes() // Should not panic
	span.End()           // Should not panic
}

func TestTracerWrapper_SpanMethodsSafe(t *testing.T) {
	// All span methods should work without panics
	wrapper := NewTracerWrapper(nil, "test-component")
	ctx := context.Background()

	_, span := wrapper.StartSpan(ctx, "test-operation", trace.SpanKindInternal)
	defer span.End()

	// These should all be safe without nil-checks
	span.SetName("new-name")
	span.RecordError(nil)
	span.AddEvent("test-event")
	span.IsRecording() // Should return false for noop
	span.SpanContext()
}

func TestTracerWrapper_Tracer(t *testing.T) {
	// Tracer accessor should return non-nil tracer
	wrapper := NewTracerWrapper(nil, "test-component")

	tracer := wrapper.Tracer()
	if tracer == nil {
		t.Fatal("expected non-nil tracer from accessor")
	}
}

func TestTracerWrapper_ContextPropagation(t *testing.T) {
	// Context should properly propagate span
	wrapper := NewTracerWrapper(nil, "test-component")
	ctx := context.Background()

	ctx, parentSpan := wrapper.StartSpan(ctx, "parent", trace.SpanKindServer)
	defer parentSpan.End()

	// Child span should work with the modified context
	_, childSpan := wrapper.StartSpan(ctx, "child", trace.SpanKindClient)
	defer childSpan.End()

	// Both spans should be valid
	if parentSpan == nil || childSpan == nil {
		t.Fatal("expected non-nil spans")
	}
}
