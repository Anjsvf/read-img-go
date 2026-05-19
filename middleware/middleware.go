package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "default-src 'none'")
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		c.Next()
	}
}

func RequestTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		timer := time.AfterFunc(d, func() {
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error_code":        "TIMEOUT",
				"error_description": "Request timed out",
			})
		})
		defer timer.Stop()
		c.Next()
	}
}

func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// ─── JWT ──────────────────────────────────────────────────────────────────────

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error_code":        "UNAUTHORIZED",
				"error_description": "Authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error_code":        "UNAUTHORIZED",
				"error_description": "Format: Bearer <token>",
			})
			return
		}

		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error_code":        "UNAUTHORIZED",
				"error_description": "Invalid or expired token",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error_code":        "UNAUTHORIZED",
				"error_description": "Invalid token claims",
			})
			return
		}

		c.Set("customer_code", claims["customer_code"])
		c.Set("claims", claims)
		c.Next()
	}
}

// ─── Rate Limiting ────────────────────────────────────────────────────────────

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.Mutex
)

func init() {
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 5*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Every(time.Second), 20)
		visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !getVisitor(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error_code":        "RATE_LIMIT_EXCEEDED",
				"error_description": "Too many requests, slow down",
			})
			return
		}
		c.Next()
	}
}
