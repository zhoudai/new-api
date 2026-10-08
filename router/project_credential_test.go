package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestProjectCredentialRoutesRequireAdmin(t *testing.T) {
	router := gin.New()
	SetApiRouter(router)
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/project-credentials"},
		{"GET", "/api/project-credentials/audits"},
		{"POST", "/api/project-credentials"},
		{"POST", "/api/project-credentials/project-a/rotate"},
		{"POST", "/api/project-credentials/project-a/disable"},
		{"POST", "/api/project-credentials/verify"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
		router.ServeHTTP(response, request)
		assert.Equal(t, http.StatusUnauthorized, response.Code, tc.path)
	}
}
