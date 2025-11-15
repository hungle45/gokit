package middleware

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hungle45/gokit/log"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

var bodyLogMethods = []string{http.MethodPost, http.MethodPut, http.MethodPatch}

const DefaultMaxResponseBodySize = 1024 * 1024 // 1 MB

type HttpServerLoggerConfig struct {
	SkipPaths           []string
	EnableHeaderLog     bool
	MaxResponseBodySize int
}

type responseCapture struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func newResponseCapture(w gin.ResponseWriter) *responseCapture {
	return &responseCapture{ResponseWriter: w}
}

func (r *responseCapture) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseCapture) BodyString(size int) string {
	n := lo.Min([]int{r.body.Len(), size})
	return r.body.String()[:n]
}

func NewDefaultHttpServerLogger() gin.HandlerFunc {
	return NewHttpServerLogger(HttpServerLoggerConfig{})
}

func NewHttpServerLogger(cfg HttpServerLoggerConfig) gin.HandlerFunc {
	maxSize := lo.Ternary(cfg.MaxResponseBodySize > 0, cfg.MaxResponseBodySize, DefaultMaxResponseBodySize)
	skipPatterns := compilePatterns(cfg.SkipPaths)

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if shouldSkipPath(path, skipPatterns) {
			c.Next()
			return
		}

		resp := newResponseCapture(c.Writer)
		c.Writer = resp
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		log.NewLoggerCtx(c, log.For(c))

		fields := []zap.Field{
			log.String("method", c.Request.Method),
			log.String("path", path),
			log.String("query", c.Request.URL.RawQuery),
			log.String("handler", extractHandlerName(c.HandlerName())),
			log.String("client_ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			log.String("latency", latency.String()),
			log.Int("status", c.Writer.Status()),
			zap.Int("response_size", c.Writer.Size()),
			log.String("response", resp.BodyString(maxSize)),
		}

		if cfg.EnableHeaderLog {
			fields = append(fields, zap.Any("headers", getHeaderMap(c.Request.Header)))
		}

		if lo.Contains(bodyLogMethods, c.Request.Method) {
			if bodyBytes, err := c.GetRawData(); err == nil {
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				fields = append(fields, zap.ByteString("body", bodyBytes))
			} else {
				fields = append(fields, zap.Any("get_body_error", err))
			}
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.Reflect("error", c.Errors.JSON()))
			log.For(c).Error(path, fields...)
		} else {
			log.For(c).Info(path, fields...)
		}
	}
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, pat := range patterns {
		compiled = append(compiled, regexp.MustCompile(pat))
	}
	return compiled
}

func shouldSkipPath(path string, patterns []*regexp.Regexp) bool {
	for _, r := range patterns {
		if r.MatchString(path) {
			return true
		}
	}
	return false
}

func getHeaderMap(header http.Header) map[string]string {
	m := make(map[string]string, len(header))
	for k, v := range header {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

func extractHandlerName(handlerName string) string {
	base := filepath.Base(handlerName)
	parts := strings.Split(base, ".")
	last := base
	if len(parts) > 0 {
		last = strings.TrimSuffix(parts[len(parts)-1], "-fm")
	}
	return last
}
