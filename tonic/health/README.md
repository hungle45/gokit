# Tonic Health

The `tonic/health` package provides HTTP handlers to expose application liveness and service health checks.

Key handlers:

- `New(checkers []check.Checker, opts ...ConfigOption) gin.HandlerFunc` — builds a health check handler that runs provided checkers concurrently and returns an aggregated JSON response. Returns HTTP 200 when all services are UP and HTTP 503 when any service is DOWN.
- `Default() gin.HandlerFunc` — shorthand for `New(nil)` (no external service checks).
- `Ping() gin.HandlerFunc` — a simple `GET` handler that returns `{ "message": "pong" }`.

Response shape for `New`:

```json
{
  "status": "UP" | "DOWN",
  "services": { "name": { "status": "UP"|"DOWN", "details": { ... } } },
  "details": { ... }
}
```

Example (from project `main.go`):

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/hungle45/gokit/tonic/health"
    "github.com/hungle45/gokit/tonic/health/check"
)

func main() {
    r := gin.Default()

    // build your checkers (redis, kafka, db, env, etc.)
    var checkers []check.Checker

    // health endpoint
    r.GET("/health", health.New(checkers))
    r.GET("/ping", health.Ping())

    r.Run(":8080")
}
```

Configuration

Use the config options to customize HTTP status codes returned by the health handler:

- `WithStatusOK(code int)` — set HTTP status code for healthy (default 200)
- `WithStatusNotOK(code int)` — set HTTP status code for unhealthy (default 503)

Notes

- Checkers must implement the `check.Checker` interface (`Name() string` and `Check(context.Context) ServiceStatus`).
- The handler runs all checkers concurrently and aggregates results.
