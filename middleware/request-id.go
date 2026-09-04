package middleware

import (
	"context"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func RequestId() func(c *gin.Context) {
	return func(c *gin.Context) {
		id := common.NewRequestId()
		// bridge 将项目请求号映射到该请求头，保留后才能跨项目回读平台审计。
		if requestedID := strings.TrimSpace(c.GetHeader(common.RequestIdKey)); requestedID != "" && len(requestedID) <= 64 {
			id = requestedID
		}
		c.Set(common.RequestIdKey, id)
		ctx := context.WithValue(c.Request.Context(), common.RequestIdKey, id)
		c.Request = c.Request.WithContext(ctx)
		c.Header(common.RequestIdKey, id)
		c.Next()
	}
}
