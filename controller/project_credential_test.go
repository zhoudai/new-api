package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestProjectCredentialAPIOnlyReturnsSecretOnIssuance(t *testing.T) {
	user := setupSSOControllerTest(t)
	// 非法登录令牌的 SQL 不应将测试凭证明文写入测试输出。
	model.DB = model.DB.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, model.DB.AutoMigrate(&model.ProjectCredential{}, &model.ProjectCredentialAudit{}))
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/project-credentials", strings.NewReader(`{"project_id":"project-api"}`))
	c.Set("id", user.Id)
	ChangeProjectCredential(c)
	require.Equal(t, http.StatusOK, response.Code)
	var payload struct {
		Data struct {
			Secret string `json:"secret"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.NotEmpty(t, payload.Data.Secret)
	for _, handler := range []gin.HandlerFunc{ListProjectCredentials, ListProjectCredentialAudits} {
		listed := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(listed)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/project-credentials", nil)
		handler(ctx)
		require.Equal(t, http.StatusOK, listed.Code)
		assert.NotContains(t, listed.Body.String(), payload.Data.Secret)
		assert.NotContains(t, listed.Body.String(), "secret_hash")
	}
	verified := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(verified)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/project-credentials/verify", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+payload.Data.Secret)
	ctx.Request.Header.Set("X-AI-Project-Id", "project-api")
	VerifyProjectCredential(ctx)
	ctx.Writer.WriteHeaderNow()
	assert.Equal(t, http.StatusNoContent, verified.Code)
	assert.Empty(t, verified.Body.String())

	// 项目凭证不是管理员登录令牌，不能用于签发或查询管理凭证。
	router := gin.New()
	router.POST("/api/project-credentials", middleware.AdminAuth(), ChangeProjectCredential)
	request := httptest.NewRequest(http.MethodPost, "/api/project-credentials", strings.NewReader(`{"project_id":"unauthorized"}`))
	request.Header.Set("Authorization", "Bearer "+payload.Data.Secret)
	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, request)
	assert.Equal(t, http.StatusUnauthorized, denied.Code)
}
