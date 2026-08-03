package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// A browser sends Origin on every POST, including same-origin ones. When the
// allowlist did not name the deployment's own origin, gin-contrib/cors answered
// 403 before the handler ran — and the login page reported that as "Invalid
// email or password", hiding the real cause.
func TestCORSAllowsConfiguredOriginAndRejectsOthers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		origin   string
		wantCode int
	}{
		{"configured deployment origin", "http://43.173.29.42:6900", http.StatusOK},
		{"configured dev origin", "http://localhost:5173", http.StatusOK},
		{"unlisted origin", "http://evil.example.com", http.StatusForbidden},
		{"no origin header at all", "", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(CORSMiddleware([]string{"http://localhost:5173", "http://43.173.29.42:6900"}))
			r.POST("/api/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Errorf("origin %q: got HTTP %d, want %d", tc.origin, w.Code, tc.wantCode)
			}
		})
	}
}
