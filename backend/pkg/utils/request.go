package utils

import (
	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/model"
)

// RequestMetaOf copies the request details the activity log stores.
//
// Call it while the request is still being served: *gin.Context is recycled
// once the handler returns, so these values have to be taken out before any
// goroutine receives them. ClientIP reads X-Forwarded-For, which nginx sets in
// front of this service, so it yields the caller's address rather than the
// proxy's.
func RequestMetaOf(c *gin.Context) model.RequestMeta {
	return model.RequestMeta{
		IP:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
}
