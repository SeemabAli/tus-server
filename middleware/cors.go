package middleware

import "net/http"

// CORSMiddleware handles CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers to allow cross-origin requests
		w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust the origin if you want to restrict to a specific domain
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, POST, PATCH, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type") // Include 'session' and 'email'
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}
