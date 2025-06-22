# EMSG Client SDK - Integration Tests

This directory contains integration tests for the EMSG Client SDK that test the SDK against real or mock EMSG servers.

## Test Categories

### 1. Mock Server Tests (`integration_test.go`)
These tests run against a mock HTTP server and test:
- User registration flow
- Message sending and validation
- System message functionality
- Retry logic configuration
- Hook functionality
- DNS resolution structure

**Run with:**
```bash
go test ./integration/
```

### 2. Docker Integration Tests (`docker_test.go`)
These tests are designed to run against a real EMSG daemon in Docker and include:
- Real server communication
- Retry logic with actual rate limiting
- Concurrent request handling
- Performance benchmarking

## Running Integration Tests

### Basic Mock Tests
```bash
# Run all mock server tests
go test ./integration/ -v

# Run specific test
go test ./integration/ -run TestMessageSending -v
```

### Docker Integration Tests
```bash
# Test against local Docker EMSG daemon
INTEGRATION_TEST=docker go test ./integration/ -run TestWithDockerEMSGDaemon -v

# Test retry logic
INTEGRATION_TEST=retry go test ./integration/ -run TestRetryWithRealServer -v

# Test concurrent requests
INTEGRATION_TEST=concurrent go test ./integration/ -run TestConcurrentRequests -v

# Test performance
INTEGRATION_TEST=performance go test ./integration/ -run TestPerformance -v
```

### Real Server Tests
```bash
# Test against sandipwalke.com (or other real EMSG server)
INTEGRATION_TEST=real go test ./integration/ -run TestWithRealEMSGServer -v
```

## Setting Up Docker EMSG Daemon

To run the Docker integration tests, you need a running EMSG daemon. Here's how to set it up:

### Option 1: Using Docker Compose
Create a `docker-compose.yml` file:

```yaml
version: '3.8'
services:
  emsg-daemon:
    image: emsg/daemon:latest  # Replace with actual EMSG daemon image
    ports:
      - "8765:8765"
    environment:
      - EMSG_DOMAIN=localhost
      - EMSG_PORT=8765
    volumes:
      - ./config:/etc/emsg
```

Run with:
```bash
docker-compose up -d
```

### Option 2: Direct Docker Run
```bash
docker run -d \
  --name emsg-daemon \
  -p 8765:8765 \
  -e EMSG_DOMAIN=localhost \
  -e EMSG_PORT=8765 \
  emsg/daemon:latest
```

### Option 3: Local Development Server
If you have the EMSG daemon source code:

```bash
# Build and run locally
cd /path/to/emsg-daemon
go build -o emsg-daemon ./cmd/daemon
./emsg-daemon --domain=localhost --port=8765
```

## DNS Configuration for Testing

For local testing, you may need to set up DNS records or use a mock resolver:

### Mock DNS (Recommended for CI/CD)
The integration tests include a mock DNS resolver that can be configured to point to your test server.

### Local DNS Setup
Add to `/etc/hosts` (Linux/macOS) or `C:\Windows\System32\drivers\etc\hosts` (Windows):
```
127.0.0.1 emsg.localhost
```

Add DNS TXT record for `_emsg.localhost`:
```
_emsg.localhost. IN TXT "https://emsg.localhost:8765"
```

## Test Environment Variables

| Variable | Values | Description |
|----------|--------|-------------|
| `INTEGRATION_TEST` | `docker`, `real`, `retry`, `concurrent`, `performance` | Enables specific integration test suites |
| `EMSG_TEST_DOMAIN` | Domain name | Override default test domain |
| `EMSG_TEST_TIMEOUT` | Duration | Override default test timeout |

## Expected Test Results

### Mock Server Tests
- ✅ All tests should pass
- ✅ Tests run quickly (< 5 seconds)
- ✅ No external dependencies

### Docker Integration Tests
- ⚠️ May fail if EMSG daemon is not running
- ⚠️ May return 404 errors (expected if endpoints differ)
- ✅ Should test authentication and message structure

### Real Server Tests
- ⚠️ May fail due to network issues
- ⚠️ May be rate limited
- ✅ Tests actual protocol implementation

## Troubleshooting

### Common Issues

#### "connection refused" errors
- Ensure EMSG daemon is running
- Check port configuration (default: 8765)
- Verify firewall settings

#### "DNS resolution failed" errors
- Check DNS configuration
- Verify `_emsg.domain.com` TXT records
- Use mock resolver for testing

#### "HTTP 404" errors
- Expected if server uses different API endpoints
- Indicates network connectivity is working
- Check server API documentation

#### "rate limit exceeded" errors
- Expected behavior for retry tests
- Indicates rate limiting is working
- Wait before retrying

### Debug Mode
Enable verbose logging:
```bash
go test ./integration/ -v -args -debug
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Integration Tests
on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    services:
      emsg-daemon:
        image: emsg/daemon:latest
        ports:
          - 8765:8765
        env:
          EMSG_DOMAIN: localhost
          EMSG_PORT: 8765

    steps:
    - uses: actions/checkout@v2
    - uses: actions/setup-go@v2
      with:
        go-version: 1.21

    - name: Run Mock Tests
      run: go test ./integration/ -v

    - name: Run Docker Tests
      run: INTEGRATION_TEST=docker go test ./integration/ -run TestWithDockerEMSGDaemon -v

    - name: Run Performance Tests
      run: INTEGRATION_TEST=performance go test ./integration/ -run TestPerformance -v
```

## Contributing

When adding new integration tests:

1. **Mock tests** should be fast and reliable
2. **Docker tests** should handle server unavailability gracefully
3. **Real server tests** should be optional and well-documented
4. Use environment variables to control test execution
5. Include proper cleanup and error handling
6. Document expected failures and their meanings

## Test Coverage

The integration tests cover:

- ✅ User registration flow
- ✅ Message creation, signing, and validation
- ✅ System message functionality
- ✅ DNS resolution
- ✅ HTTP authentication
- ✅ Retry logic and rate limiting
- ✅ Concurrent request handling
- ✅ Performance characteristics
- ✅ Hook functionality
- ✅ Error handling and edge cases

These tests complement the unit tests in the `/test` directory and provide end-to-end validation of the SDK functionality.
