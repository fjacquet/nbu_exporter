# Requirements Document

## Introduction

This specification addresses code quality improvements identified during analysis of the OpenTelemetry integration and related codebase. The improvements focus on reducing code duplication, improving maintainability, enhancing readability, and following Go best practices without changing existing functionality.

## Glossary

- **System**: The NBU Exporter application
- **Span**: OpenTelemetry tracing unit representing an operation
- **Tracer**: OpenTelemetry component that creates spans
- **Attribute**: Key-value metadata attached to spans
- **DRY**: Don't Repeat Yourself principle
- **Godoc**: Go documentation comments

## Requirements

### Requirement 1: Reduce Code Duplication

**User Story:** As a developer, I want to eliminate duplicate span creation logic, so that the codebase is easier to maintain and modify.

#### Acceptance Criteria

1. WHEN multiple span creation helper functions exist with identical implementations, THE System SHALL consolidate them into a single reusable function
2. WHEN the consolidated function is called with different operation names, THE System SHALL create spans with the appropriate operation name
3. WHEN the tracer is nil, THE System SHALL return the original context without creating a span
4. WHERE span creation is needed, THE System SHALL use the consolidated helper function instead of duplicated code

### Requirement 2: Centralize Configuration Constants

**User Story:** As a developer, I want span attribute keys centralized in one location, so that I can maintain consistency and avoid typos.

#### Acceptance Criteria

1. THE System SHALL define all span attribute keys as package-level constants
2. WHEN recording span attributes, THE System SHALL use the defined constants instead of string literals
3. THE System SHALL organize constants by category (HTTP, NetBackup, Scrape)
4. WHERE new attributes are added, THE System SHALL add them to the constants file

### Requirement 3: Improve Error Message Maintainability

**User Story:** As a developer, I want complex error messages extracted to constants, so that they are easier to update and maintain.

#### Acceptance Criteria

1. WHEN error messages exceed 5 lines, THE System SHALL extract them to package-level constants or templates
2. THE System SHALL use format strings with placeholders for dynamic values
3. WHEN error messages are updated, THE System SHALL require changes in only one location
4. THE System SHALL maintain the same error message content and formatting

### Requirement 4: Enhance Configuration Validation

**User Story:** As a user, I want OpenTelemetry endpoint validation to check format correctness, so that I receive clear feedback about configuration errors.

#### Acceptance Criteria

1. WHEN OpenTelemetry is enabled, THE System SHALL validate the endpoint format as "host:port"
2. WHEN the endpoint port is invalid, THE System SHALL return a descriptive error message
3. THE System SHALL validate port numbers are within the range 1-65535
4. WHERE endpoint validation fails, THE System SHALL prevent application startup

### Requirement 5: Improve Test Consistency

**User Story:** As a developer, I want consistent error handling in tests, so that test intent is clear and maintainable.

#### Acceptance Criteria

1. WHEN tests expect errors, THE System SHALL explicitly check for error presence
2. WHEN tests allow graceful degradation, THE System SHALL document this with comments
3. THE System SHALL not silently ignore errors with underscore assignment without justification
4. WHERE error handling differs between tests, THE System SHALL document the reasoning

### Requirement 6: Extract Complex Conditionals

**User Story:** As a developer, I want complex conditional logic extracted to named functions, so that code intent is clearer.

#### Acceptance Criteria

1. WHEN conditional expressions exceed 2 conditions, THE System SHALL extract them to named functions
2. THE System SHALL use descriptive function names that explain the condition's purpose
3. WHEN reading the code, THE System SHALL make the business logic immediately apparent
4. WHERE conditionals are extracted, THE System SHALL maintain identical behavior

### Requirement 7: Optimize Span Attribute Recording

**User Story:** As a developer, I want span attributes batched in single calls, so that performance is optimized.

#### Acceptance Criteria

1. WHEN recording multiple span attributes, THE System SHALL batch them in a single SetAttributes call
2. THE System SHALL create attribute slices before calling SetAttributes
3. WHEN span is nil, THE System SHALL skip attribute recording without errors
4. THE System SHALL maintain the same attribute values and types

### Requirement 8: Improve Documentation Completeness

**User Story:** As a developer, I want complete godoc comments on exported functions, so that API usage is clear.

#### Acceptance Criteria

1. THE System SHALL document all parameters with their purpose and constraints
2. THE System SHALL document all return values with their meaning
3. THE System SHALL include usage examples for complex functions
4. WHERE functions have side effects, THE System SHALL document them clearly

### Requirement 9: Fix Test Function Naming Convention

**User Story:** As a developer, I want test function names to follow Go conventions and pass SonarCloud quality checks, so that code quality standards are met.

#### Acceptance Criteria

1. THE System SHALL name test functions using only alphanumeric characters without underscores
2. WHEN test functions are named, THE System SHALL follow the pattern `Test<TypeName><MethodName>`
3. THE System SHALL maintain test readability by using descriptive subtest names with `t.Run()`
4. WHERE test functions are renamed, THE System SHALL preserve all test logic and behavior
5. THE System SHALL pass SonarCloud "Sonar Way" quality profile checks for Go test naming

### Requirement 10: Eliminate Duplicate String Literals

**User Story:** As a developer, I want duplicate string literals extracted to constants, so that code is maintainable and passes SonarCloud quality checks.

#### Acceptance Criteria

1. WHEN a string literal is used more than 3 times, THE System SHALL extract it to a named constant
2. THE System SHALL define string constants at package level or in a constants file
3. THE System SHALL use descriptive constant names that indicate the string's purpose
4. WHERE string literals represent configuration keys or field names, THE System SHALL extract them to constants
5. THE System SHALL pass SonarCloud "Sonar Way" quality profile rule go:S1192 for string literal duplication

### Requirement 11: Reduce Cognitive Complexity

**User Story:** As a developer, I want functions with high cognitive complexity refactored into smaller functions, so that code is easier to understand and maintain.

#### Acceptance Criteria

1. WHEN a function has cognitive complexity above 15, THE System SHALL refactor it into smaller helper functions
2. THE System SHALL extract nested conditional logic into well-named helper functions
3. THE System SHALL extract loop bodies into separate functions when they contain complex logic
4. WHERE functions are refactored, THE System SHALL maintain identical behavior and test coverage
5. THE System SHALL pass SonarCloud cognitive complexity threshold of 15 for all functions

### Requirement 12: Centralize Test Helper Functions

**User Story:** As a developer, I want duplicate test helper functions consolidated in a shared package, so that test code is maintainable and follows DRY principles.

#### Acceptance Criteria

1. WHEN test helper functions are duplicated across multiple test files, THE System SHALL consolidate them into a shared testutil package
2. THE System SHALL provide fluent builder interfaces for common test objects (Config, MockServer)
3. WHEN creating test configurations, THE System SHALL use the TestConfigBuilder instead of manual struct initialization
4. WHEN creating mock HTTP servers, THE System SHALL use the MockServerBuilder instead of inline httptest.NewServer calls
5. THE System SHALL provide a LoadTestData helper for loading test fixtures from files

### Requirement 13: Enhance Error Context

**User Story:** As a developer, I want error messages to include sufficient context for debugging, so that I can quickly identify and resolve issues.

#### Acceptance Criteria

1. WHEN HTTP requests fail, THE System SHALL include the request URL in error messages
2. WHEN JSON unmarshaling fails, THE System SHALL include the HTTP status code and content-type in error messages
3. WHEN non-JSON responses are received, THE System SHALL include a preview of the response body in error messages
4. THE System SHALL provide actionable debugging information in all error messages

### Requirement 14: Complete Package Documentation

**User Story:** As a developer, I want comprehensive package-level documentation, so that I understand the purpose and usage of each package.

#### Acceptance Criteria

1. THE System SHALL provide package-level documentation for all internal packages
2. WHEN documenting packages, THE System SHALL describe key components and their relationships
3. THE System SHALL include usage examples in package documentation
4. WHERE design patterns are used, THE System SHALL document them in package comments
