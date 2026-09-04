package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIdPreservesBridgeRequestId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestId())
	router.GET("/", func(c *gin.Context) {
		c.Status(204)
	})

	requestID := "project-request-1"
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(common.RequestIdKey, requestID)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, 204, recorder.Code)
	assert.Equal(t, requestID, recorder.Header().Get(common.RequestIdKey))
}
