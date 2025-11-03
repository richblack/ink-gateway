package server

import (
	"log"
	"net/http"
	"semantic-text-processor/services"
	"strings"
	"time"
)

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapper, r)
		
		duration := time.Since(start)
		
		// Use structured logger if available
		if s.services.Logger != nil {
			s.services.Logger.Info("HTTP request",
				services.String("method", r.Method),
				services.String("path", r.URL.Path),
				services.String("remote_addr", r.RemoteAddr),
				services.Int("status_code", wrapper.statusCode),
				services.Duration("duration", duration),
				services.String("user_agent", r.UserAgent()),
			)
		} else {
			// Fallback to standard logging
			log.Printf("%s %s %d %v %s", 
				r.Method, 
				r.URL.Path, 
				wrapper.statusCode, 
				duration,
				r.RemoteAddr,
			)
		}
	})
}

// corsMiddleware handles CORS headers with enhanced support for Obsidian
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Allow specific origins including Obsidian's app:// protocol
		allowedOrigins := []string{
			"app://obsidian.md",
			"capacitor://localhost",
			"http://localhost",
			"https://localhost",
		}
		
		// Check if origin is allowed or allow all for development
		originAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed || strings.HasPrefix(origin, allowed) {
				originAllowed = true
				break
			}
		}
		
		if originAllowed || origin == "" {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*") // Fallback for development
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// contentTypeMiddleware sets default content type
func (s *Server) contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "" && (r.Method == "POST" || r.Method == "PUT") {
			r.Header.Set("Content-Type", "application/json")
		}
		next.ServeHTTP(w, r)
	})
}

// performanceMiddleware tracks request performance metrics
func (s *Server) performanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapper, r)
		
		duration := time.Since(start)
		
		// Record metrics if service is available
		if s.services.MetricsService != nil {
			tags := map[string]string{
				"method":     r.Method,
				"endpoint":   r.URL.Path,
				"status":     http.StatusText(wrapper.statusCode),
				"status_code": string(rune(wrapper.statusCode)),
			}
			
			// Record request duration
			s.services.MetricsService.RecordDuration("http.request.duration", duration, tags)
			
			// Increment request counter
			s.services.MetricsService.IncrementCounter("http.requests.total", tags)
			
			// Track error rates
			if wrapper.statusCode >= 400 {
				s.services.MetricsService.IncrementCounter("http.requests.errors", tags)
			}
			
			// Track slow requests (>1 second)
			if duration > time.Second {
				s.services.MetricsService.IncrementCounter("http.requests.slow", tags)
			}
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}