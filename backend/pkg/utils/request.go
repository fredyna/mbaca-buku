package utils

import (
	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/model"
)

// RequestMetaOf copies the request details the activity log stores.
//
// Call it while the request is still being served: *gin.Context is recycled
// once the handler returns, so these values have to be taken out before any
// goroutine receives them. ClientIP's result depends on how the engine was
// configured — see ConfigureTrustedProxies, which the server calls once at
// startup so this always yields the caller's real address rather than one it
// dictated itself.
func RequestMetaOf(c *gin.Context) model.RequestMeta {
	return model.RequestMeta{
		IP:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
}

// trustedProxyRanges are the private-network ranges gin will treat as this
// service's reverse proxy. docker-compose.yml does not publish the backend's
// port and defines no custom network, so nginx and the backend only ever
// talk over Docker's default bridge network, which is carved out of
// 172.16.0.0/12 — nothing outside that range is a proxy this service put
// there itself.
var trustedProxyRanges = []string{"172.16.0.0/12"}

// ConfigureTrustedProxies locks down where gin will accept a forwarded
// client IP from. RequestMetaOf above stores that address as a security-audit
// column, so a value the caller can dictate is worse than none.
//
// gin's zero-value trusts every proxy (0.0.0.0/0, ::/0). Under that setting,
// Context.ClientIP() walks X-Forwarded-For right-to-left, treats every hop
// as trusted, and returns the LEFTMOST entry — whatever the client put
// there. nginx (nginx/default.conf.template) only *appends* to
// X-Forwarded-For via $proxy_add_x_forwarded_for, so a forged leading value
// survives the trip untouched; but nginx *replaces* X-Real-Ip with
// $remote_addr on every request, so that header can never carry a caller's
// lie through nginx.
//
// Restricting SetTrustedProxies to trustedProxyRanges and reading only
// X-Real-Ip means gin only trusts that header when the request's immediate
// TCP peer is inside the Docker network nginx runs on. A request that
// reaches this service any other way — including one that skips nginx
// entirely — falls back to the real connecting address instead of a spoofed
// header, forged or not.
func ConfigureTrustedProxies(r *gin.Engine) error {
	if err := r.SetTrustedProxies(trustedProxyRanges); err != nil {
		return err
	}
	r.RemoteIPHeaders = []string{"X-Real-Ip"}
	return nil
}
