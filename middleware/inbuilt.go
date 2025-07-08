package middleware

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	json "github.com/json-iterator/go"
	"github.com/xraph/forgerouter"
	"golang.org/x/time/rate"
)

// Logger Built-in middleware
var Logger = forgerouter.Logger

var Recoverer = forgerouter.Recoverer

var Timeout = forgerouter.Timeout

// =============================================================================
// CORS Middleware
// =============================================================================

type CORSConfig struct {
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
	ExposedHeaders     []string
	AllowCredentials   bool
	MaxAge             int
	AllowOriginFunc    func(origin string) bool
	OptionsPassthrough bool
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"},
		AllowedHeaders: []string{
			"Accept", "Authorization", "Content-Type", "X-CSRF-Token",
			"X-Requested-With", "X-API-Key", "X-Request-ID",
		},
		ExposedHeaders:     []string{},
		AllowCredentials:   false,
		MaxAge:             86400, // 24 hours
		OptionsPassthrough: false,
	}
}

func CORS(config ...CORSConfig) forgerouter.MiddlewareFunc {
	cfg := DefaultCORSConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowOrigin := ""
			if cfg.AllowOriginFunc != nil {
				if cfg.AllowOriginFunc(origin) {
					allowOrigin = origin
				}
			} else {
				for _, allowedOrigin := range cfg.AllowedOrigins {
					if allowedOrigin == "*" || allowedOrigin == origin {
						allowOrigin = allowedOrigin
						break
					}
				}
			}

			// Set CORS headers
			if allowOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			}

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if len(cfg.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))

				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}

				if !cfg.OptionsPassthrough {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Opinionated CORS middleware
func OpinionatedCORS(config ...CORSConfig) forgerouter.OpinionatedMiddleware {
	cfg := DefaultCORSConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return forgerouter.NewMiddleware("cors").
		Description("Cross-Origin Resource Sharing (CORS) middleware").
		Before(func(ctx *forgerouter.MiddlewareContext) error {
			origin := ctx.Request.Header.Get("Origin")

			// Check if origin is allowed
			allowOrigin := ""
			if cfg.AllowOriginFunc != nil {
				if cfg.AllowOriginFunc(origin) {
					allowOrigin = origin
				}
			} else {
				for _, allowedOrigin := range cfg.AllowedOrigins {
					if allowedOrigin == "*" || allowedOrigin == origin {
						allowOrigin = allowedOrigin
						break
					}
				}
			}

			// Set CORS headers
			if allowOrigin != "" {
				ctx.Response.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			}

			if cfg.AllowCredentials {
				ctx.Response.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if len(cfg.ExposedHeaders) > 0 {
				ctx.Response.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
			}

			// Handle preflight requests
			if ctx.Request.Method == "OPTIONS" {
				ctx.Response.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
				ctx.Response.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))

				if cfg.MaxAge > 0 {
					ctx.Response.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}

				if !cfg.OptionsPassthrough {
					ctx.StatusCode = http.StatusNoContent
					ctx.Processed = true
					return nil
				}
			}

			return nil
		}).
		CachingSafe().
		Build()
}

// =============================================================================
// Rate Limiting Middleware
// =============================================================================

type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
	KeyFunc           func(*http.Request) string
	OnLimitReached    func(w http.ResponseWriter, r *http.Request)
	SkipFunc          func(*http.Request) bool
	Store             RateLimitStore
}

type RateLimitStore interface {
	GetLimiter(key string) *rate.Limiter
	CleanupExpired()
}

type inMemoryRateLimitStore struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewInMemoryRateLimitStore(rps float64, burst int) RateLimitStore {
	store := &inMemoryRateLimitStore{
		limiters: make(map[string]*rateLimiterEntry),
		rps:      rate.Limit(rps),
		burst:    burst,
	}

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			store.CleanupExpired()
		}
	}()

	return store
}

func (s *inMemoryRateLimitStore) GetLimiter(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.limiters[key]
	if !exists {
		entry = &rateLimiterEntry{
			limiter:  rate.NewLimiter(s.rps, s.burst),
			lastSeen: time.Now(),
		}
		s.limiters[key] = entry
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

func (s *inMemoryRateLimitStore) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiry := time.Now().Add(-time.Hour) // Remove entries older than 1 hour
	for key, entry := range s.limiters {
		if entry.lastSeen.Before(expiry) {
			delete(s.limiters, key)
		}
	}
}

func defaultRateLimitKeyFunc(r *http.Request) string {
	// Try to get real IP from various headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}

	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func RateLimit(config RateLimitConfig) forgerouter.MiddlewareFunc {
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 10 // Default 10 RPS
	}
	if config.BurstSize <= 0 {
		config.BurstSize = int(config.RequestsPerSecond) * 2 // Default burst is 2x RPS
	}
	if config.KeyFunc == nil {
		config.KeyFunc = defaultRateLimitKeyFunc
	}
	if config.Store == nil {
		config.Store = NewInMemoryRateLimitStore(config.RequestsPerSecond, config.BurstSize)
	}
	if config.OnLimitReached == nil {
		config.OnLimitReached = func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.SkipFunc != nil && config.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := config.KeyFunc(r)
			limiter := config.Store.GetLimiter(key)

			if !limiter.Allow() {
				config.OnLimitReached(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Opinionated Rate Limiting middleware
func OpinionatedRateLimit(config RateLimitConfig) forgerouter.OpinionatedMiddleware {
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 10
	}
	if config.BurstSize <= 0 {
		config.BurstSize = int(config.RequestsPerSecond) * 2
	}
	if config.KeyFunc == nil {
		config.KeyFunc = defaultRateLimitKeyFunc
	}
	if config.Store == nil {
		config.Store = NewInMemoryRateLimitStore(config.RequestsPerSecond, config.BurstSize)
	}

	return forgerouter.NewMiddleware("rate_limit").
		Description("Rate limiting middleware to prevent abuse").
		Before(func(ctx *forgerouter.MiddlewareContext) error {
			if config.SkipFunc != nil && config.SkipFunc(ctx.Request) {
				return nil
			}

			key := config.KeyFunc(ctx.Request)
			limiter := config.Store.GetLimiter(key)

			if !limiter.Allow() {
				return forgerouter.TooManyRequests("Rate limit exceeded")
			}

			return nil
		}).
		AddResponse("429", "Rate limit exceeded").
		Build()
}

// =============================================================================
// Request ID Middleware
// =============================================================================

type RequestIDConfig struct {
	HeaderName    string
	Generator     func() string
	ForceGenerate bool
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

func RequestID(config ...RequestIDConfig) forgerouter.MiddlewareFunc {
	cfg := RequestIDConfig{
		HeaderName:    "X-Request-ID",
		Generator:     generateRequestID,
		ForceGenerate: false,
	}
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(cfg.HeaderName)

			if requestID == "" || cfg.ForceGenerate {
				requestID = cfg.Generator()
			}

			// Set request ID in context
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			r = r.WithContext(ctx)

			// Set response header
			w.Header().Set(cfg.HeaderName, requestID)

			next.ServeHTTP(w, r)
		})
	}
}

// Opinionated Request ID middleware
func OpinionatedRequestID(config ...RequestIDConfig) forgerouter.OpinionatedMiddleware {
	cfg := RequestIDConfig{
		HeaderName:    "X-Request-ID",
		Generator:     generateRequestID,
		ForceGenerate: false,
	}
	if len(config) > 0 {
		cfg = config[0]
	}

	return forgerouter.NewMiddleware("request_id").
		Description("Generates and tracks unique request identifiers").
		Before(func(ctx *forgerouter.MiddlewareContext) error {
			requestID := ctx.Request.Header.Get(cfg.HeaderName)

			if requestID == "" || cfg.ForceGenerate {
				requestID = cfg.Generator()
			}

			// Set request ID in context and middleware context
			ctx.RequestID = requestID
			ctx.Metadata["request_id"] = requestID

			// Set in request context
			reqCtx := context.WithValue(ctx.Request.Context(), "request_id", requestID)
			ctx.Request = ctx.Request.WithContext(reqCtx)

			// Set response header
			ctx.Response.Header().Set(cfg.HeaderName, requestID)

			return nil
		}).
		AddHeader("X-Request-ID", "Unique request identifier", false).
		CachingSafe().
		Build()
}

// =============================================================================
// JWT Authentication Middleware
// =============================================================================

type JWTConfig struct {
	SigningKey     interface{}
	SigningMethod  jwt.SigningMethod
	TokenLookup    string // "header:Authorization" or "query:token" or "cookie:jwt"
	AuthScheme     string
	SkipFunc       func(*http.Request) bool
	ErrorHandler   func(w http.ResponseWriter, r *http.Request, err error)
	SuccessHandler func(w http.ResponseWriter, r *http.Request, token *jwt.Token)
	Claims         jwt.Claims
}

func DefaultJWTConfig() JWTConfig {
	return JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		TokenLookup:   "header:Authorization",
		AuthScheme:    "Bearer",
		Claims:        jwt.MapClaims{},
	}
}

func extractTokenFromRequest(r *http.Request, config JWTConfig) (string, error) {
	parts := strings.Split(config.TokenLookup, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token lookup format")
	}

	switch parts[0] {
	case "header":
		auth := r.Header.Get(parts[1])
		if auth == "" {
			return "", fmt.Errorf("missing authorization header")
		}

		if config.AuthScheme != "" {
			prefix := config.AuthScheme + " "
			if !strings.HasPrefix(auth, prefix) {
				return "", fmt.Errorf("invalid authorization scheme")
			}
			return auth[len(prefix):], nil
		}
		return auth, nil

	case "query":
		token := r.URL.Query().Get(parts[1])
		if token == "" {
			return "", fmt.Errorf("missing token in query")
		}
		return token, nil

	case "cookie":
		cookie, err := r.Cookie(parts[1])
		if err != nil {
			return "", fmt.Errorf("missing token in cookie: %v", err)
		}
		return cookie.Value, nil

	default:
		return "", fmt.Errorf("unsupported token lookup method")
	}
}

func JWT(config JWTConfig) forgerouter.MiddlewareFunc {
	if config.SigningKey == nil {
		panic("JWT middleware requires signing key")
	}
	if config.SigningMethod == nil {
		config.SigningMethod = jwt.SigningMethodHS256
	}
	if config.TokenLookup == "" {
		config.TokenLookup = "header:Authorization"
	}
	if config.AuthScheme == "" {
		config.AuthScheme = "Bearer"
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.SkipFunc != nil && config.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			tokenString, err := extractTokenFromRequest(r, config)
			if err != nil {
				config.ErrorHandler(w, r, err)
				return
			}

			token, err := jwt.ParseWithClaims(tokenString, config.Claims, func(token *jwt.Token) (interface{}, error) {
				if token.Method != config.SigningMethod {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return config.SigningKey, nil
			})

			if err != nil {
				config.ErrorHandler(w, r, err)
				return
			}

			if !token.Valid {
				config.ErrorHandler(w, r, fmt.Errorf("invalid token"))
				return
			}

			// Store token in context
			ctx := context.WithValue(r.Context(), "jwt_token", token)
			ctx = context.WithValue(ctx, "jwt_claims", token.Claims)
			r = r.WithContext(ctx)

			if config.SuccessHandler != nil {
				config.SuccessHandler(w, r, token)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Opinionated JWT middleware
func OpinionatedJWT(config JWTConfig) forgerouter.OpinionatedMiddleware {
	if config.SigningKey == nil {
		panic("JWT middleware requires signing key")
	}
	if config.SigningMethod == nil {
		config.SigningMethod = jwt.SigningMethodHS256
	}
	if config.TokenLookup == "" {
		config.TokenLookup = "header:Authorization"
	}
	if config.AuthScheme == "" {
		config.AuthScheme = "Bearer"
	}

	return forgerouter.NewMiddleware("jwt_auth").
		Description("JWT token authentication middleware").
		Before(func(ctx *forgerouter.MiddlewareContext) error {
			if config.SkipFunc != nil && config.SkipFunc(ctx.Request) {
				return nil
			}

			tokenString, err := extractTokenFromRequest(ctx.Request, config)
			if err != nil {
				return forgerouter.Unauthorized("Missing or invalid authorization token: " + err.Error())
			}

			token, err := jwt.ParseWithClaims(tokenString, config.Claims, func(token *jwt.Token) (interface{}, error) {
				if token.Method != config.SigningMethod {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return config.SigningKey, nil
			})

			if err != nil {
				return forgerouter.Unauthorized("Invalid token: " + err.Error())
			}

			if !token.Valid {
				return forgerouter.Unauthorized("Token is not valid")
			}

			// Store token in context and middleware context
			reqCtx := context.WithValue(ctx.Request.Context(), "jwt_token", token)
			reqCtx = context.WithValue(reqCtx, "jwt_claims", token.Claims)
			ctx.Request = ctx.Request.WithContext(reqCtx)

			ctx.Metadata["jwt_token"] = token
			ctx.Metadata["jwt_claims"] = token.Claims

			if config.SuccessHandler != nil {
				config.SuccessHandler(ctx.Response, ctx.Request, token)
			}

			return nil
		}).
		RequiresAuth().
		AddSecurityRequirement(forgerouter.RequireBearer("JWTAuth")).
		AddResponse("401", "Unauthorized - Invalid or missing JWT token").
		AddHeader("Authorization", "JWT Bearer token", true).
		Build()
}

// =============================================================================
// Compression Middleware
// =============================================================================

type CompressionConfig struct {
	Level     int      // Compression level (1-9)
	MinLength int      // Minimum response size to compress
	Types     []string // MIME types to compress
}

func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 1024, // 1KB
		Types: []string{
			"text/html",
			"text/css",
			"text/javascript",
			"text/plain",
			"application/json",
			"application/javascript",
			"application/xml",
			"application/x-javascript",
		},
	}
}

type compressResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
	level  int
}

func (w *compressResponseWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

func (w *compressResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(statusCode)
}

func shouldCompress(r *http.Request, contentType string, contentLength int, config CompressionConfig) bool {
	// Check if client accepts gzip
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}

	// Check minimum length
	if contentLength > 0 && contentLength < config.MinLength {
		return false
	}

	// Check content type
	for _, t := range config.Types {
		if strings.Contains(contentType, t) {
			return true
		}
	}

	return false
}

func Compression(config ...CompressionConfig) forgerouter.MiddlewareFunc {
	cfg := DefaultCompressionConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Create a buffer to capture the response
			buf := &bytes.Buffer{}
			wrapped := &compressResponseWriter{
				ResponseWriter: w,
				writer:         buf,
				level:          cfg.Level,
			}

			// Call next handler with wrapped writer
			next.ServeHTTP(wrapped, r)

			// Check if we should compress
			contentType := w.Header().Get("Content-Type")
			if shouldCompress(r, contentType, buf.Len(), cfg) {
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Vary", "Accept-Encoding")

				gz, err := gzip.NewWriterLevel(w, cfg.Level)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer gz.Close()

				gz.Write(buf.Bytes())
			} else {
				w.Write(buf.Bytes())
			}
		})
	}
}

// =============================================================================
// Security Headers Middleware
// =============================================================================

type SecurityConfig struct {
	XSSProtection         string
	ContentTypeNosniff    bool
	XFrameOptions         string
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	HSTSPreload           bool
	ContentSecurityPolicy string
	ReferrerPolicy        string
	PermissionsPolicy     string
}

func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    true,
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionsPolicy:     "camera=(), microphone=(), geolocation=()",
	}
}

func SecureHeaders(config ...SecurityConfig) forgerouter.MiddlewareFunc {
	cfg := DefaultSecurityConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", cfg.XSSProtection)
			}

			if cfg.ContentTypeNosniff {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}

			if cfg.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", cfg.XFrameOptions)
			}

			if cfg.HSTSMaxAge > 0 && r.TLS != nil {
				hstsValue := fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)
				if cfg.HSTSIncludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				if cfg.HSTSPreload {
					hstsValue += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			if cfg.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}

			if cfg.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}

			if cfg.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// =============================================================================
// Request Logging Middleware
// =============================================================================

type LoggingConfig struct {
	Logger       *log.Logger
	Skip         func(*http.Request) bool
	Format       string
	TimeFormat   string
	UTC          bool
	CustomFields map[string]func(*http.Request, time.Duration) interface{}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int64
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += int64(size)
	return size, err
}

func RequestLogging(config ...LoggingConfig) forgerouter.MiddlewareFunc {
	cfg := LoggingConfig{
		Logger:     log.Default(),
		Format:     "${time} ${method} ${path} ${status} ${size} ${duration}",
		TimeFormat: time.RFC3339,
		UTC:        false,
	}
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			recorder := &responseRecorder{
				ResponseWriter: w,
				status:         200,
			}

			next.ServeHTTP(recorder, r)

			duration := time.Since(start)

			// Build log entry
			logData := map[string]interface{}{
				"time":     start.Format(cfg.TimeFormat),
				"method":   r.Method,
				"path":     r.URL.Path,
				"status":   recorder.status,
				"size":     recorder.size,
				"duration": duration.String(),
				"ip":       defaultRateLimitKeyFunc(r),
			}

			// Add custom fields
			if cfg.CustomFields != nil {
				for key, fn := range cfg.CustomFields {
					logData[key] = fn(r, duration)
				}
			}

			// Log as JSON
			logJSON, _ := json.Marshal(logData)
			cfg.Logger.Println(string(logJSON))
		})
	}
}

// =============================================================================
// Body Size Limit Middleware
// =============================================================================

type BodyLimitConfig struct {
	Limit    int64
	SkipFunc func(*http.Request) bool
}

func BodyLimit(limit int64, config ...BodyLimitConfig) forgerouter.MiddlewareFunc {
	cfg := BodyLimitConfig{
		Limit: limit,
	}
	if len(config) > 0 {
		cfg = config[0]
		if cfg.Limit == 0 {
			cfg.Limit = limit
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.SkipFunc != nil && cfg.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			if r.ContentLength > cfg.Limit {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, cfg.Limit)
			next.ServeHTTP(w, r)
		})
	}
}

// =============================================================================
// Circuit Breaker Middleware
// =============================================================================

type CircuitBreakerConfig struct {
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
	IsSuccessful  func(err error) bool
}

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	isSuccessful  func(err error) bool
	onStateChange func(name string, from State, to State)

	mutex      sync.Mutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
}

func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          "circuit_breaker",
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		isSuccessful:  config.IsSuccessful,
		onStateChange: config.OnStateChange,
	}

	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}
	if cb.interval <= 0 {
		cb.interval = time.Minute
	}
	if cb.timeout <= 0 {
		cb.timeout = 60 * time.Second
	}
	if cb.readyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	}
	if cb.isSuccessful == nil {
		cb.isSuccessful = func(err error) bool {
			return err == nil
		}
	}

	cb.toNewGeneration(time.Now())
	return cb
}

func (cb *CircuitBreaker) Execute(req func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	err = req()
	cb.afterRequest(generation, cb.isSuccessful(err))
	return err
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, fmt.Errorf("circuit breaker is open")
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, fmt.Errorf("circuit breaker is half-open and max requests exceeded")
	}

	cb.counts.Requests++
	return generation, nil
}

func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
	case StateHalfOpen:
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.TotalFailures++
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

func CircuitBreakerMiddleware(config CircuitBreakerConfig) forgerouter.MiddlewareFunc {
	cb := NewCircuitBreaker(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := cb.Execute(func() error {
				recorder := &responseRecorder{ResponseWriter: w, status: 200}
				next.ServeHTTP(recorder, r)

				if recorder.status >= 500 {
					return fmt.Errorf("server error: %d", recorder.status)
				}
				return nil
			})

			if err != nil {
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			}
		})
	}
}

// =============================================================================
// Metrics Middleware
// =============================================================================

type MetricsConfig struct {
	Namespace   string
	Subsystem   string
	SkipFunc    func(*http.Request) bool
	GroupedPath func(string) string
}

type Metrics struct {
	requestCount    map[string]*uint64
	requestDuration map[string]*uint64
	requestSize     map[string]*uint64
	responseSize    map[string]*uint64
	mutex           sync.RWMutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		requestCount:    make(map[string]*uint64),
		requestDuration: make(map[string]*uint64),
		requestSize:     make(map[string]*uint64),
		responseSize:    make(map[string]*uint64),
	}
}

func (m *Metrics) Record(method, path string, status int, duration time.Duration, requestSize, responseSize int64) {
	key := fmt.Sprintf("%s_%s_%d", method, path, status)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.requestCount[key] == nil {
		m.requestCount[key] = new(uint64)
		m.requestDuration[key] = new(uint64)
		m.requestSize[key] = new(uint64)
		m.responseSize[key] = new(uint64)
	}

	atomic.AddUint64(m.requestCount[key], 1)
	atomic.AddUint64(m.requestDuration[key], uint64(duration.Nanoseconds()))
	atomic.AddUint64(m.requestSize[key], uint64(requestSize))
	atomic.AddUint64(m.responseSize[key], uint64(responseSize))
}

func (m *Metrics) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]interface{})

	for key, count := range m.requestCount {
		stats[key] = map[string]interface{}{
			"count":          atomic.LoadUint64(count),
			"total_duration": atomic.LoadUint64(m.requestDuration[key]),
			"total_req_size": atomic.LoadUint64(m.requestSize[key]),
			"total_res_size": atomic.LoadUint64(m.responseSize[key]),
		}
	}

	return stats
}

func MetricsMiddleware(metrics *Metrics, config ...MetricsConfig) forgerouter.MiddlewareFunc {
	cfg := MetricsConfig{}
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.SkipFunc != nil && cfg.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			recorder := &responseRecorder{ResponseWriter: w, status: 200}

			next.ServeHTTP(recorder, r)

			duration := time.Since(start)
			path := r.URL.Path
			if cfg.GroupedPath != nil {
				path = cfg.GroupedPath(path)
			}

			requestSize := r.ContentLength
			if requestSize < 0 {
				requestSize = 0
			}

			metrics.Record(r.Method, path, recorder.status, duration, requestSize, recorder.size)
		})
	}
}
