package middleware

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	ratelimit "github.com/zhaojiewen/open-station/internal/infrastructure/persistence/redis"
	"github.com/zhaojiewen/open-station/pkg/config"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

var jsonContentTypes = map[string]bool{
	"application/json":                  true,
	"application/json; charset=utf-8":   true,
	"application/json;charset=utf-8":    true,
	"text/event-stream":                 true, // SSE streaming
}

func SafeMiddleware(service *ratelimit.SafeService, cfg *config.SafeConfig) gin.HandlerFunc {
	allowedMethods := make(map[string]bool, len(cfg.AllowedMethods))
	for _, m := range cfg.AllowedMethods {
		allowedMethods[strings.ToUpper(strings.TrimSpace(m))] = true
	}

	blockedUAs := make(map[string]bool, len(cfg.BlockedUserAgents))
	for _, ua := range cfg.BlockedUserAgents {
		blockedUAs[strings.ToLower(strings.TrimSpace(ua))] = true
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		// 1. IP whitelist - fast path, skip all checks
		if service.IsWhitelisted(clientIP) {
			c.Next()
			return
		}

		// 2. Request method validation
		if len(allowedMethods) > 0 && !allowedMethods[c.Request.Method] {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error":   apperrors.ErrMethodNotAllowed.Code,
				"message": apperrors.ErrMethodNotAllowed.Message,
			})
			c.Abort()
			return
		}

		// 3. URL length validation
		if cfg.MaxURLLength > 0 && len(c.Request.URL.Path) > cfg.MaxURLLength {
			c.JSON(http.StatusRequestURITooLong, gin.H{
				"error":   apperrors.ErrURLTooLong.Code,
				"message": apperrors.ErrURLTooLong.Message,
			})
			c.Abort()
			return
		}

		if cfg.MaxQueryLength > 0 && len(c.Request.URL.RawQuery) > cfg.MaxQueryLength {
			c.JSON(http.StatusRequestURITooLong, gin.H{
				"error":   apperrors.ErrURLTooLong.Code,
				"message": apperrors.ErrURLTooLong.Message,
			})
			c.Abort()
			return
		}

		// 4. Path traversal detection
		if cfg.PathTraversalCheck {
			if detectPathTraversal(c.Request.URL.Path) || detectPathTraversal(c.Request.URL.RawQuery) {
				logger.Warn("safe: path traversal detected",
					zap.String("ip", clientIP),
					zap.String("path", c.Request.URL.Path))
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   apperrors.ErrPathTraversal.Code,
					"message": apperrors.ErrPathTraversal.Message,
				})
				c.Abort()
				return
			}
		}

		// 5. Content-Type enforcement
		if cfg.EnforceContentType && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH") {
			contentType := c.GetHeader("Content-Type")
			if !jsonContentTypes[strings.ToLower(contentType)] {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error":   apperrors.ErrInvalidContentType.Code,
					"message": apperrors.ErrInvalidContentType.Message,
				})
				c.Abort()
				return
			}
		}

		// 6. User-agent validation
		userAgent := c.GetHeader("User-Agent")
		if cfg.BlockEmptyUserAgent && userAgent == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   apperrors.ErrBadUserAgent.Code,
				"message": apperrors.ErrBadUserAgent.Message,
			})
			c.Abort()
			return
		}
		if len(blockedUAs) > 0 && blockedUAs[strings.ToLower(userAgent)] {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   apperrors.ErrBadUserAgent.Code,
				"message": apperrors.ErrBadUserAgent.Message,
			})
			c.Abort()
			return
		}

		// 7. Suspicious header detection
		if cfg.BlockSuspiciousHeaders {
			if detectSuspiciousHeaders(c.Request, cfg.MaxSingleHeaderKB) {
				logger.Warn("safe: suspicious headers detected",
					zap.String("ip", clientIP))
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   apperrors.ErrSuspiciousHeader.Code,
					"message": apperrors.ErrSuspiciousHeader.Message,
				})
				c.Abort()
				return
			}
		}

		// 8. Header size validation
		if cfg.MaxHeaderSizeKB > 0 {
			totalSize := 0
			for k, vals := range c.Request.Header {
				totalSize += len(k)
				for _, v := range vals {
					totalSize += len(v)
				}
			}
			if totalSize > cfg.MaxHeaderSizeKB*1024 {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error":   apperrors.ErrRequestHeadersTooLarge.Code,
					"message": apperrors.ErrRequestHeadersTooLarge.Message,
				})
				c.Abort()
				return
			}
		}

		// 9. IP blacklist/block check (static + Redis auto-block)
		blocked, err := service.IsBlocked(c.Request.Context(), clientIP)
		if err != nil {
			logger.Warn("safe: failed to check IP block status",
				zap.String("ip", clientIP), zap.Error(err))
		} else if blocked {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   apperrors.ErrIPBlocked.Code,
				"message": apperrors.ErrIPBlocked.Message,
			})
			c.Abort()
			return
		}

		// 10. Body size limit
		if cfg.BodySizeLimitMB > 0 {
			limit := int64(cfg.BodySizeLimitMB) * 1024 * 1024
			if c.Request.ContentLength > limit {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error":   apperrors.ErrRequestBodyTooLarge.Code,
					"message": apperrors.ErrRequestBodyTooLarge.Message,
				})
				c.Abort()
				return
			}
			c.Request.Body = io.NopCloser(io.LimitReader(c.Request.Body, limit))
		}

		// 11. Concurrent connection limiting
		if cfg.MaxConcurrentConns > 0 {
			allowed, release, err := service.AcquireConnection(c.Request.Context(), clientIP, cfg.MaxConcurrentConns)
			if err != nil {
				logger.Warn("safe: failed to check concurrent connections",
					zap.String("ip", clientIP), zap.Error(err))
			} else if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":   apperrors.ErrTooManyConcurrentConns.Code,
					"message": apperrors.ErrTooManyConcurrentConns.Message,
				})
				c.Abort()
				return
			} else {
				defer release()
			}
		}

		// 12. IP rate limiting with burst auto-block and violation tracking
		if err := service.CheckIPRateLimit(c.Request.Context(), clientIP, cfg.IPRateLimit.RPS, cfg.IPRateLimit.Burst); err != nil {
			// Burst auto-block: check if this IP is massively exceeding the limit
			if cfg.BurstAutoBlock.Enabled {
				count, cntErr := service.GetIPRateCount(c.Request.Context(), clientIP)
				if cntErr != nil {
					logger.Warn("safe: failed to check IP burst count",
						zap.String("ip", clientIP), zap.Error(cntErr))
				} else if count > cfg.IPRateLimit.RPS*cfg.BurstAutoBlock.BurstFactor {
					_ = service.AutoBlockIP(c.Request.Context(), clientIP, cfg.BurstAutoBlock.BlockDurationS)
					logger.Warn("safe: IP auto-blocked due to burst attack",
						zap.String("ip", clientIP), zap.Int("count", count))
					c.JSON(http.StatusForbidden, gin.H{
						"error":   apperrors.ErrBurstAttackAutoBlocked.Code,
						"message": apperrors.ErrBurstAttackAutoBlocked.Message,
					})
					c.Abort()
					return
				}
			}

			// Track rate violations for repeated-offender auto-block
			if cfg.RateViolationBlock.Enabled {
				wasBlocked, recErr := service.RecordRateViolation(
					c.Request.Context(), clientIP,
					cfg.RateViolationBlock.WindowS,
					cfg.RateViolationBlock.MaxViolations,
					cfg.RateViolationBlock.BlockDurationS,
				)
				if recErr != nil {
					logger.Warn("safe: failed to record rate violation",
						zap.String("ip", clientIP), zap.Error(recErr))
				} else if wasBlocked {
					logger.Warn("safe: IP auto-blocked due to repeated rate limit violations",
						zap.String("ip", clientIP))
					c.JSON(http.StatusForbidden, gin.H{
						"error":   apperrors.ErrRateViolationBlocked.Code,
						"message": apperrors.ErrRateViolationBlocked.Message,
					})
					c.Abort()
					return
				}
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   apperrors.ErrIPRateLimitExceeded.Code,
				"message": apperrors.ErrIPRateLimitExceeded.Message,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func detectPathTraversal(s string) bool {
	if s == "" {
		return false
	}

	lower := strings.ToLower(s)

	if strings.Contains(lower, "..") {
		return true
	}

	if strings.Contains(lower, "%2e%2e") || strings.Contains(lower, "%2e.") || strings.Contains(lower, ".%2e") {
		return true
	}

	if strings.Contains(s, "%00") || strings.Contains(s, "\x00") {
		return true
	}

	return false
}

func detectSuspiciousHeaders(r *http.Request, maxSingleHeaderKB int) bool {
	// Check for CRLF injection in header values
	for _, vals := range r.Header {
		for _, v := range vals {
			if strings.Contains(v, "\r") || strings.Contains(v, "\n") {
				return true
			}
		}
	}

	// Check for overly long single header values
	if maxSingleHeaderKB > 0 {
		for _, vals := range r.Header {
			for _, v := range vals {
				if len(v) > maxSingleHeaderKB*1024 {
					return true
				}
			}
		}
	}

	// Check for suspicious X-Forwarded-For (internal IP spoofing attempt)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		for _, part := range strings.Split(xff, ",") {
			ip := strings.TrimSpace(part)
			if isInternalIP(ip) {
				return true
			}
		}
	}

	// Check custom X-Forwarded-For header aliases commonly used in attacks
	suspiciousHeaders := []string{
		"X-Real-IP",
		"X-Client-IP",
		"X-Originating-IP",
		"X-Remote-IP",
	}
	seen := make(map[string]bool)
	for _, h := range suspiciousHeaders {
		if v := r.Header.Get(h); v != "" {
			canonical := strings.ToLower(h)
			if seen[canonical] {
				return true
			}
			seen[canonical] = true
		}
	}

	return false
}

func isInternalIP(ip string) bool {
	// Detect attempts to spoof loopback or private network IPs
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return true
	}
	if strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "172.16.") ||
		strings.HasPrefix(ip, "172.17.") ||
		strings.HasPrefix(ip, "172.18.") ||
		strings.HasPrefix(ip, "172.19.") ||
		strings.HasPrefix(ip, "172.20.") ||
		strings.HasPrefix(ip, "172.21.") ||
		strings.HasPrefix(ip, "172.22.") ||
		strings.HasPrefix(ip, "172.23.") ||
		strings.HasPrefix(ip, "172.24.") ||
		strings.HasPrefix(ip, "172.25.") ||
		strings.HasPrefix(ip, "172.26.") ||
		strings.HasPrefix(ip, "172.27.") ||
		strings.HasPrefix(ip, "172.28.") ||
		strings.HasPrefix(ip, "172.29.") ||
		strings.HasPrefix(ip, "172.30.") ||
		strings.HasPrefix(ip, "172.31.") ||
		strings.HasPrefix(ip, "0.") ||
		strings.HasPrefix(ip, "169.254.") {
		return true
	}
	return false
}
