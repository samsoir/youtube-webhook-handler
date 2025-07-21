# Testing Guide

This document covers all testing procedures and requirements for the YouTube Webhook Service.

## Overview

The project includes comprehensive unit tests for all webhook functionality with **87.8% code coverage**.

## Quick Testing Commands

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run tests with verbose output
make test-verbose

# Run benchmark tests
make benchmark

# Security scanning
make security-scan

# Test local function startup
make test-local
```

## Test Coverage

### Current Status
- **Current Coverage**: 87.8% (maintained automatically via CI)
- **Minimum Required**: 85% (builds fail below this threshold)

### Coverage by Component

- **Main Handler (`YouTubeWebhook`)**: 100% - All HTTP methods and CORS handling
- **Verification Challenge**: 100% - YouTube subscription challenge responses
- **Notification Processing**: 87.5% - XML parsing, filtering, GitHub API calls
- **GitHub Integration**: 85.7% - Repository dispatch events and error handling
- **Video Filtering**: 92.9% - New video detection logic
- **Error Handling**: 100% - HTTP response write error handling

## Continuous Integration

The project uses GitHub Actions with automatic coverage validation:
- **All pushes and PRs** trigger comprehensive test suites
- **Coverage below 85%** fails the build automatically
- **Security scanning** with Gosec and govulncheck
- **Multiple quality gates** including linting and formatting

### CI Workflows

1. **`ci.yml`**: Main testing workflow with Test, Lint, Security Scan, Terraform Validate
2. **`deploy.yml`**: Production deployment workflow (runs after CI success)

## Test Categories

### Unit Tests

**Webhook Request Handling:**
- Verification challenge responses
- CORS preflight handling
- HTTP method validation
- Invalid request handling

**YouTube Integration:**
- XML feed parsing
- Video metadata extraction
- New video detection logic
- Timestamp validation

**GitHub Integration:**
- Repository dispatch API calls
- Payload formatting
- Authentication handling
- Error response processing

**Error Scenarios:**
- Invalid XML parsing
- Missing environment variables
- GitHub API failures
- HTTP response write errors

### Integration Tests

**Local Function Testing:**
- Function startup validation
- End-to-end request processing
- Environment variable handling

**Benchmark Tests:**
- Performance validation
- Memory usage optimization
- Response time measurement

## Writing New Tests

### Test Structure

Tests are organized in the `function/` directory:
```
function/
├── webhook.go          # Main implementation
├── webhook_test.go     # Comprehensive test suite
├── go.mod             # Go module dependencies
└── go.sum             # Dependency checksums
```

### Test Requirements

1. **Coverage**: New code must maintain 85%+ coverage
2. **Edge Cases**: Test error conditions and malformed inputs
3. **Isolation**: Tests must not affect production systems
4. **Mocking**: Use mocks for external GitHub API calls

### Example Test

```go
func TestWebhook_WriteErrors(t *testing.T) {
    tests := []struct {
        name   string
        method string
        url    string
        body   string
    }{
        {"Method not allowed write error", "DELETE", "/", ""},
        {"Challenge write error", "GET", "/?hub.challenge=test", ""},
        {"Bad request write error", "POST", "/", "invalid body"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
            w := &failingResponseWriter{ResponseRecorder: httptest.NewRecorder()}
            
            // This should not panic even with write errors
            assert.NotPanics(t, func() {
                YouTubeWebhook(w, req)
            })
        })
    }
}
```

## Running Tests Locally

### Prerequisites

```bash
# Set up development environment
make setup

# Navigate to function directory
cd function
```

### Development Testing Workflow

```bash
# Quick test run during development
make test

# Watch for changes and run tests automatically (if available)
find . -name "*.go" | entr -c make test

# Full validation before committing
make test-coverage
```

## Security Testing

### Vulnerability Scanning

```bash
# Run comprehensive security scan
make security-scan

# This includes:
# - govulncheck: Known vulnerability detection
# - gosec: Static analysis security scanner
```

### Security Test Coverage

- **G104 (CWE-703)**: Unhandled errors - All HTTP response writes properly handled
- **Dependency vulnerabilities**: All dependencies scanned and updated
- **Authentication**: GitHub token validation and error handling
- **Input validation**: XML parsing with malformed input handling

## Performance Testing

### Benchmark Tests

```bash
# Run performance benchmarks
make benchmark

# Example output:
# BenchmarkWebhookProcessing-8    1000    1234567 ns/op    1024 B/op    10 allocs/op
```

### Performance Targets

- **Response Time**: < 500ms for typical webhook processing
- **Memory Usage**: < 50MB memory footprint
- **Concurrency**: Handle multiple concurrent webhook requests

## Test Environment

Tests run in isolated environments and use mocked external services to prevent:
- Production GitHub API calls
- Network dependencies
- Authentication requirements
- Rate limiting issues

### Mock GitHub Server

The test suite includes a comprehensive mock GitHub server that:
- Simulates GitHub API responses
- Tracks received payloads
- Tests authentication handling
- Validates request formatting

## Troubleshooting

### Common Issues

**Tests failing with import errors:**
```bash
# Ensure dependencies are downloaded
cd function && go mod download
```

**Coverage reports not generating:**
```bash
# Run with explicit coverage
cd function && go test -coverprofile=coverage.out ./...
cd function && go tool cover -html=coverage.out
```

**Local function startup issues:**
```bash
# Check environment variables
echo $GITHUB_TOKEN
echo $REPO_OWNER
echo $REPO_NAME

# Test with minimal setup
make test-local
```

### Test Data

Tests use predictable test data to ensure consistent results:
- Fixed timestamps for video detection logic
- Mock GitHub API responses
- Controlled XML feed structures

## Performance

- **Test Suite Runtime**: ~1-2 seconds
- **Coverage Generation**: ~2-3 seconds  
- **CI Pipeline**: ~45-60 seconds (including Terraform validation)

## Best Practices

1. **Test Names**: Use descriptive names explaining the scenario
2. **Assertions**: Use specific assertions with clear error messages
3. **Setup/Teardown**: Clean up test state between runs
4. **Mocking**: Mock GitHub API consistently across tests
5. **Edge Cases**: Test boundary conditions and error scenarios
6. **Documentation**: Document complex test scenarios and edge cases

## Coverage Targets

| Component | Target Coverage | Current |
|-----------|----------------|---------|
| Webhook Handler | 95%+ | 100% |
| GitHub Integration | 85%+ | 85.7% |
| Video Processing | 90%+ | 92.9% |
| Overall Project | 85%+ | 87.8% |

## Related Documentation

- [Contributing Guide](CONTRIBUTING.md) - Development workflow
- [README](README.md) - Main project documentation
- [GitHub Actions](.github/workflows/) - CI/CD configuration
- [Makefile](Makefile) - Available test commands