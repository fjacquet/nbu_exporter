package telemetry

// This file defines error message templates for common failure scenarios.
// Templates provide consistent, actionable error messages with troubleshooting steps.
//
// Using templates instead of inline error messages:
//   - Centralizes error message maintenance
//   - Ensures consistent formatting and content
//   - Makes it easier to update troubleshooting steps
//   - Reduces code duplication
//
// Usage:
//
//	if resp.StatusCode() == http.StatusNotAcceptable {
//	    return fmt.Errorf(telemetry.ErrAPIVersionNotSupportedTemplate,
//	        apiVersion, apiVersion, url)
//	}
//
// Each template includes:
//   - Clear description of the error
//   - Explanation of common causes
//   - Step-by-step troubleshooting instructions
//   - Example configuration or commands
//   - Relevant context (URL, status code, etc.)

// Error message templates for common scenarios
const (
	// ErrAPIVersionNotSupportedTemplate is returned when the NetBackup server doesn't support the configured API version
	ErrAPIVersionNotSupportedTemplate = `API version %s is not supported by the NetBackup server (HTTP 406 Not Acceptable).

The server may be running a version of NetBackup that does not support API version %s.

Supported API versions:
  - 3.0  (NetBackup 10.0-10.4)
  - 12.0 (NetBackup 10.5)
  - 13.0 (NetBackup 11.0)

Troubleshooting steps:
1. Verify your NetBackup server version: bpgetconfig -g | grep VERSION
2. Update the 'apiVersion' field in config.yaml to match your server version
3. Or remove the 'apiVersion' field to enable automatic version detection

Example configuration:
  nbuserver:
    apiVersion: "12.0"  # For NetBackup 10.5
    # Or omit apiVersion for automatic detection

Request URL: %s`

	// ErrNonJSONResponseTemplate is returned when the server returns non-JSON content
	ErrNonJSONResponseTemplate = `NetBackup server returned non-JSON response (Content-Type: %s).

This usually indicates:
1. Wrong API endpoint URL (check 'uri' in config.yaml)
2. Authentication failure (verify API key is valid)
3. Server configuration issue (check NetBackup REST API is enabled)

Request URL: %s
Response preview: %s`
)
