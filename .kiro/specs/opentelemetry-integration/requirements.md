# Requirements Document

## Introduction

This specification defines the integration of OpenTelemetry (OTel) observability capabilities into the NBU Exporter application. The integration will add distributed tracing for NetBackup API calls and enhanced structured logging with trace correlation, while preserving the existing Prometheus metrics functionality. This enhancement will enable operators to diagnose performance issues, track request flows through the exporter, and correlate logs with traces for improved troubleshooting.

## Glossary

- **NBU Exporter**: The NetBackup Prometheus exporter application that collects backup infrastructure metrics
- **OpenTelemetry (OTel)**: A vendor-neutral observability framework for collecting traces, metrics, and logs
- **Trace**: A record of the path of a request through distributed systems
- **Span**: A single unit of work within a trace, representing an operation
- **Trace Context**: Metadata that identifies and correlates spans within a trace
- **OTLP**: OpenTelemetry Protocol for transmitting telemetry data
- **Exporter**: A component that sends telemetry data to a backend system
- **Collector**: The OpenTelemetry Collector that receives, processes, and exports telemetry
- **Instrumentation**: Code that generates telemetry data from application operations
- **Scrape Cycle**: A single iteration of metric collection triggered by Prometheus
- **NetBackup API**: The REST API provided by Veritas NetBackup for data retrieval
- **Prometheus Collector**: The component implementing prometheus.Collector interface
- **Trace Propagation**: The mechanism for passing trace context across service boundaries

## Requirements

### Requirement 1

**User Story:** As a NetBackup operator, I want to enable optional OpenTelemetry tracing so that I can diagnose slow scrapes and identify performance bottlenecks in API calls without impacting existing Prometheus functionality.

#### Acceptance Criteria

1. WHEN the operator configures OpenTelemetry settings in the config file, THE NBU Exporter SHALL initialize the OpenTelemetry tracer provider with the specified OTLP endpoint
2. WHEN OpenTelemetry is not configured, THE NBU Exporter SHALL operate normally with existing Prometheus metrics without any tracing overhead
3. WHEN the tracer provider is initialized, THE NBU Exporter SHALL use OTLP gRPC exporter to send trace data to the configured collector endpoint
4. WHEN the application shuts down, THE NBU Exporter SHALL flush all pending traces before terminating
5. WHERE OpenTelemetry is enabled, THE NBU Exporter SHALL set the service name to "nbu-exporter" in trace metadata

### Requirement 2

**User Story:** As a NetBackup operator, I want distributed tracing for NetBackup API calls so that I can identify which specific API endpoints are slow and understand the complete request flow.

#### Acceptance Criteria

1. WHEN a Prometheus scrape occurs, THE NBU Exporter SHALL create a root span named "prometheus.scrape" for the entire collection cycle
2. WHEN FetchStorage is called, THE NBU Exporter SHALL create a child span named "netbackup.fetch_storage" with the storage API endpoint as an attribute
3. WHEN FetchAllJobs is called, THE NBU Exporter SHALL create a child span named "netbackup.fetch_jobs" with the jobs API endpoint and time window as attributes
4. WHEN FetchJobDetails is called for pagination, THE NBU Exporter SHALL create a child span named "netbackup.fetch_job_page" with the offset and page number as attributes
5. WHEN an HTTP request is made to NetBackup API, THE NBU Exporter SHALL record the HTTP method, URL, status code, and response time as span attributes
6. IF an API call fails, THEN THE NBU Exporter SHALL record the error message and mark the span status as error
7. WHEN a span completes, THE NBU Exporter SHALL record the duration in the span timing data

### Requirement 3

**User Story:** As a NetBackup operator, I want structured logging with trace correlation so that I can link log messages to specific traces and understand the context of errors.

#### Acceptance Criteria

1. WHEN OpenTelemetry is enabled, THE NBU Exporter SHALL replace logrus with OpenTelemetry's log bridge for structured logging
2. WHEN a log message is emitted within a traced operation, THE NBU Exporter SHALL include the trace ID and span ID in the log record
3. WHEN a log message is emitted, THE NBU Exporter SHALL include standard attributes such as timestamp, severity level, and source location
4. WHEN an error occurs during API calls, THE NBU Exporter SHALL log the error with trace context for correlation
5. WHERE OpenTelemetry is disabled, THE NBU Exporter SHALL continue using logrus for logging without trace correlation

### Requirement 4

**User Story:** As a NetBackup operator, I want configurable OpenTelemetry settings so that I can control tracing behavior, sampling rates, and export destinations without code changes.

#### Acceptance Criteria

1. THE NBU Exporter SHALL support an optional "opentelemetry" section in the YAML configuration file
2. WHEN the "opentelemetry.enabled" field is set to true, THE NBU Exporter SHALL initialize OpenTelemetry instrumentation
3. WHEN the "opentelemetry.enabled" field is set to false or omitted, THE NBU Exporter SHALL disable OpenTelemetry features
4. THE NBU Exporter SHALL accept "opentelemetry.endpoint" configuration to specify the OTLP collector endpoint
5. THE NBU Exporter SHALL accept "opentelemetry.insecure" configuration to control TLS verification for the OTLP connection
6. THE NBU Exporter SHALL accept "opentelemetry.samplingRate" configuration to control the percentage of traces sampled
7. WHERE "opentelemetry.samplingRate" is set to 1.0, THE NBU Exporter SHALL trace all scrape cycles
8. WHERE "opentelemetry.samplingRate" is set to 0.1, THE NBU Exporter SHALL trace approximately 10% of scrape cycles

### Requirement 5

**User Story:** As a NetBackup operator, I want detailed span attributes for API operations so that I can analyze performance patterns and identify specific slow operations.

#### Acceptance Criteria

1. WHEN a storage fetch operation completes, THE NBU Exporter SHALL record the number of storage units retrieved as a span attribute
2. WHEN a job fetch operation completes, THE NBU Exporter SHALL record the number of jobs retrieved and the time window queried as span attributes
3. WHEN pagination occurs, THE NBU Exporter SHALL record the total number of pages fetched as a span attribute
4. WHEN an HTTP request is made, THE NBU Exporter SHALL record the request size and response size as span attributes
5. WHEN API version detection occurs, THE NBU Exporter SHALL create a span named "netbackup.detect_version" with attempted versions as attributes
6. WHEN a span represents an HTTP operation, THE NBU Exporter SHALL follow OpenTelemetry semantic conventions for HTTP attributes

### Requirement 6

**User Story:** As a NetBackup operator, I want OpenTelemetry integration to be backward compatible so that existing deployments continue to work without configuration changes.

#### Acceptance Criteria

1. WHEN the configuration file does not include an "opentelemetry" section, THE NBU Exporter SHALL start successfully with tracing disabled
2. WHEN OpenTelemetry is disabled, THE NBU Exporter SHALL not create any trace spans or attempt to connect to an OTLP endpoint
3. WHEN OpenTelemetry is disabled, THE NBU Exporter SHALL maintain existing Prometheus metrics functionality without performance degradation
4. WHEN OpenTelemetry initialization fails, THE NBU Exporter SHALL log a warning and continue operating with tracing disabled
5. THE NBU Exporter SHALL not require any changes to existing Prometheus scrape configurations

### Requirement 7

**User Story:** As a NetBackup operator, I want trace context propagation so that I can correlate traces across multiple systems if the exporter is part of a larger observability pipeline.

#### Acceptance Criteria

1. WHEN OpenTelemetry is enabled, THE NBU Exporter SHALL configure W3C Trace Context propagation as the default propagator
2. WHEN the exporter receives an HTTP request with trace context headers, THE NBU Exporter SHALL extract the parent trace context
3. WHEN the exporter creates spans, THE NBU Exporter SHALL propagate the trace context to child operations
4. WHEN the exporter makes HTTP requests to NetBackup API, THE NBU Exporter SHALL inject trace context into outgoing request headers
5. THE NBU Exporter SHALL support baggage propagation for cross-cutting contextual information

### Requirement 8

**User Story:** As a NetBackup operator, I want resource attributes in traces so that I can identify which exporter instance generated specific traces in multi-instance deployments.

#### Acceptance Criteria

1. WHEN OpenTelemetry is initialized, THE NBU Exporter SHALL set the service name resource attribute to "nbu-exporter"
2. WHEN OpenTelemetry is initialized, THE NBU Exporter SHALL set the service version resource attribute to the application version
3. WHEN OpenTelemetry is initialized, THE NBU Exporter SHALL set the host name resource attribute to the system hostname
4. WHEN OpenTelemetry is initialized, THE NBU Exporter SHALL set the NetBackup server host as a custom resource attribute
5. WHERE multiple exporter instances monitor different NetBackup servers, THE NBU Exporter SHALL include the target NetBackup server in resource attributes for differentiation

### Requirement 9

**User Story:** As a NetBackup operator, I want minimal performance overhead from tracing so that OpenTelemetry instrumentation does not significantly impact scrape times or resource usage.

#### Acceptance Criteria

1. WHEN OpenTelemetry is enabled with sampling rate less than 1.0, THE NBU Exporter SHALL only create spans for sampled requests
2. WHEN OpenTelemetry is disabled, THE NBU Exporter SHALL have zero tracing overhead
3. WHEN spans are created, THE NBU Exporter SHALL use batch span processor to minimize export overhead
4. WHEN the OTLP endpoint is unavailable, THE NBU Exporter SHALL continue operating and log connection errors without blocking scrapes
5. THE NBU Exporter SHALL export spans asynchronously to avoid blocking metric collection

### Requirement 10

**User Story:** As a NetBackup operator, I want clear documentation and examples so that I can easily configure and use OpenTelemetry features.

#### Acceptance Criteria

1. THE NBU Exporter SHALL include example OpenTelemetry configuration in the sample config.yaml file
2. THE NBU Exporter SHALL document all OpenTelemetry configuration options in the README
3. THE NBU Exporter SHALL provide example docker-compose configuration for running with OpenTelemetry Collector
4. THE NBU Exporter SHALL document common trace queries for identifying slow API calls
5. THE NBU Exporter SHALL include troubleshooting guidance for OpenTelemetry connectivity issues
