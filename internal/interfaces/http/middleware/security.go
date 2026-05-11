package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders 安全响应头中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS - 强制HTTPS（生产环境启用）
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 防止MIME类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// XSS保护
		c.Header("X-XSS-Protection", "1; mode=block")

		// 内容安全策略（根据需求调整）
		// c.Header("Content-Security-Policy", "default-src 'self'")

		// 禁用缓存敏感信息
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Header("Pragma", "no-cache")

		// 移除可能暴露服务器信息的头
		c.Header("X-Powered-By", "")
		c.Header("Server", "")

		c.Next()
	}
}

// ForceHTTPS 强制HTTPS中间件（生产环境）
func ForceHTTPS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否通过反向代理传递的HTTPS标记
		if c.Request.Header.Get("X-Forwarded-Proto") == "http" {
			// 重定向到HTTPS
			httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
			c.Redirect(http.StatusMovedPermanently, httpsURL)
			c.Abort()
			return
		}

		// 直接检查（非反向代理环境）
		if c.Request.TLS == nil {
			// 开发环境可能不强制HTTPS
			// 生产环境应该强制
		}

		c.Next()
	}
}

// CORSMiddleware CORS中间件（根据需要配置）
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false

		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Device-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// 处理OPTIONS预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimitByIP IP级别的请求限制（用于auth端点）
func RateLimitByIP(rps int, burst int) gin.HandlerFunc {
	// 这里可以集成现有的rate limit服务
	// 或者使用专门的IP限流器

	return func(c *gin.Context) {
		// IP限流逻辑（简化示例）
		// 实际应该使用Redis进行分布式限流
		c.Next()
	}
}

// NoCache 禁用缓存的中间件
func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Expires", "0")
		c.Header("Pragma", "no-cache")
		c.Next()
	}
}