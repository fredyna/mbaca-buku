package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clientIPRouter builds an engine configured exactly the way main.go
// configures it (via ConfigureTrustedProxies) and reports whatever
// RequestMetaOf resolves the caller's IP to.
func clientIPRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	require.NoError(t, ConfigureTrustedProxies(r))
	r.GET("/whoami", func(c *gin.Context) {
		c.String(http.StatusOK, RequestMetaOf(c).IP)
	})
	return r
}

// TestConfigureTrustedProxiesRejectsForgedHeaders guards the finding that
// c.ClientIP() used to hand back whatever the caller put in
// X-Forwarded-For / X-Real-Ip, because gin's default trusts every proxy.
// A request whose immediate TCP peer is outside the trusted Docker network —
// i.e. one that did not come through nginx — must fall back to that real
// peer address no matter what its headers claim.
func TestConfigureTrustedProxiesRejectsForgedHeaders(t *testing.T) {
	tests := []struct {
		name string
		req  func() *http.Request
	}{
		{
			name: "forged X-Forwarded-For",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
				req.RemoteAddr = "8.8.8.8:12345"
				req.Header.Set("X-Forwarded-For", "6.6.6.6")
				return req
			},
		},
		{
			name: "forged X-Real-Ip",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
				req.RemoteAddr = "8.8.8.8:12345"
				req.Header.Set("X-Real-Ip", "6.6.6.6")
				return req
			},
		},
		{
			name: "both headers forged",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
				req.RemoteAddr = "8.8.8.8:12345"
				req.Header.Set("X-Forwarded-For", "6.6.6.6")
				req.Header.Set("X-Real-Ip", "6.6.6.6")
				return req
			},
		},
	}

	r := clientIPRouter(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, tc.req())

			require.Equal(t, http.StatusOK, w.Code)
			got := w.Body.String()
			assert.NotEqual(t, "6.6.6.6", got, "the caller must not be able to dictate the recorded IP")
			assert.Equal(t, "8.8.8.8", got, "an untrusted peer's real address must be used instead")
		})
	}
}

// TestConfigureTrustedProxiesHonoursNginx guards the other half of the same
// change: a request that genuinely comes from nginx (the Docker network
// ConfigureTrustedProxies trusts) must still resolve to the address nginx
// recorded in X-Real-Ip, not to nginx's own address.
func TestConfigureTrustedProxiesHonoursNginx(t *testing.T) {
	r := clientIPRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	// 172.20.0.5 stands in for nginx's address on Docker's default bridge
	// network, which falls inside trustedProxyRanges (172.16.0.0/12).
	req.RemoteAddr = "172.20.0.5:54321"
	req.Header.Set("X-Real-Ip", "203.0.113.7")
	// nginx never sets this; a stray value here must still be ignored, since
	// RemoteIPHeaders only lists X-Real-Ip.
	req.Header.Set("X-Forwarded-For", "6.6.6.6")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "203.0.113.7", w.Body.String())
}
