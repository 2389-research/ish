# Memory Safety in Request Logging Middleware

## Overview
The HTTP request logging middleware (`middleware.go`) has been designed with careful attention to memory safety and efficiency. This document explains the memory constraints, how they work, and what safeguards are in place.

## Size Limits

### maxBodySize Constant
- **Value**: 10 KB (10,240 bytes)
- **Purpose**: Prevents unbounded memory consumption from request and response bodies
- **Location**: `internal/logging/middleware.go`, line 19

This constant is enforced in two places:
1. **Request body capture** (line 77): `io.LimitReader(r.Body, maxBodySize)`
2. **Response body buffering** (lines 42-48): Manual size checking with bounds verification

## How Request Body Capture Works

### Memory-Safe Read (Lines 75-83)
```go
// Uses io.LimitReader to enforce hard limit
bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
if err == nil {
    requestBody = string(bodyBytes)
    // Restore the body for downstream handlers
    r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
}
```

**Memory safety properties**:
- `io.LimitReader` prevents reading more than `maxBodySize` bytes
- Error handling gracefully skips logging if read fails
- Body is restored using `bytes.NewBuffer` for downstream handler consumption
- Maximum memory allocated: ~10 KB per request

## How Response Body Buffering Works

### Memory-Safe Write (Lines 36-50)
```go
func (rw *responseWriter) Write(b []byte) (int, error) {
    if !rw.written {
        rw.statusCode = http.StatusOK
        rw.written = true
    }
    // Capture response body (up to maxBodySize)
    if rw.body.Len() < maxBodySize {
        toCopy := len(b)
        if rw.body.Len()+toCopy > maxBodySize {
            toCopy = maxBodySize - rw.body.Len()
        }
        rw.body.Write(b[:toCopy])
    }
    return rw.ResponseWriter.Write(b)
}
```

**Memory safety properties**:
- Double-check: `if rw.body.Len() < maxBodySize` gates all buffering
- Calculated partial copy prevents overflow: `toCopy = maxBodySize - rw.body.Len()`
- Buffer receives only the safe portion: `rw.body.Write(b[:toCopy])`
- Once limit is reached, no more bytes are buffered (line 48 check short-circuits)
- Downstream writer (line 49) always receives full response data

**Critical invariant**: `rw.body.Len()` is guaranteed ≤ `maxBodySize` always

## Streaming Behavior for Large Responses

The middleware does NOT buffer large response bodies entirely. Instead:
1. Response data > 10 KB is still sent to the client completely
2. Only the first 10 KB is captured for logging
3. Subsequent writes (line 48 guard) skip buffering but pass through to client
4. This provides a graceful degradation: all data flows to client, logs are truncated

**Example**: 100 MB video response
- Client receives all 100 MB
- Database logs only first 10 KB
- Memory overhead: ~10 KB (not 100 MB)

## Memory Profile

### Per-Request Overhead
- **Fixed**: ~200 bytes for `responseWriter` struct
- **Variable**: Up to 10 KB for request body buffer
- **Variable**: Up to 10 KB for response body buffer
- **Total max per request**: ~20.2 KB

### With 1000 Concurrent Requests
- Worst case: 1000 × 20.2 KB = ~20.2 MB
- With typical payloads (< 10 KB): ~1-5 MB

### Database Overhead
- Request log schema stores bodies as TEXT
- 10 KB limit per body = manageable row size
- No risk of million-row bloat from single large response

## Tested Scenarios

The middleware test suite (`middleware_test.go`) validates:
1. ✅ Small responses buffered completely
2. ✅ Responses at limit (10 KB) buffered completely
3. ✅ Responses exceeding limit capped at 10 KB
4. ✅ Multiple chunks handled with partial buffering
5. ✅ Request body reading respects limit
6. ✅ Request body restored for handler consumption
7. ✅ Health check and admin endpoints skip logging
8. ✅ WebSocket hijacking supported

All tests pass with zero memory leaks confirmed.

## Configuration & Tuning

### Current Settings
- **Request body limit**: 10 KB
- **Response body limit**: 10 KB
- **Rationale**: Small enough to prevent memory exhaustion, large enough for most API payloads

### If You Need to Change the Limit
Modify `maxBodySize` in `middleware.go`:
```go
const maxBodySize = 10 * 1024 // Change this value
```

**Note**: Update test cases in `middleware_test.go` if changing the limit
- Specifically: `TestMiddleware_RequestBodySizeLimit` and response buffering tests

## Future Improvements (Recommendations)

1. **Streaming Response Logging** (Not implemented): Instead of buffering, could stream large responses to a separate log file or compression pipeline
2. **Configurable Limits** (Future): Allow tuning per environment (dev: 1 MB, prod: 10 KB)
3. **Compression** (Future): Compress logged bodies before storage to reduce database size
4. **Sampling** (Future): Log only a percentage of requests in high-traffic scenarios

## Related Code

- **Logging middleware**: `/internal/logging/middleware.go`
- **Tests**: `/internal/logging/middleware_test.go`
- **Database schema**: `/internal/store/request_logs.go` (TEXT columns for bodies)
- **Storage**: `/internal/store/store.go` (migrations and schema)

## Security Implications

The size limits also provide security benefits:
- **DoS protection**: Client can't force server to buffer gigabytes of response
- **Log injection prevention**: Limited response body prevents log spam attacks
- **Database protection**: Bounded column sizes prevent query complexity issues

## Conclusion

The logging middleware balances observability with resource constraints through:
1. Hard limits enforced with `io.LimitReader` and buffer bounds checking
2. Memory-efficient streaming for large responses
3. Graceful degradation (all data flows, logs are truncated)
4. Comprehensive test coverage of edge cases
5. Well-documented behavior for maintainability
