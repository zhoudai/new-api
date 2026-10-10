package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSSOControllerTest(t *testing.T) *model.User {
	t.Helper()
	previousDB, previousType := model.DB, common.MainDatabaseType()
	previousRedis, previousSecret := common.RedisEnabled, common.SessionSecret
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.AuthFlow{}))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.SessionSecret = "sso-controller-test-secret"
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.RedisEnabled = previousRedis
		common.SessionSecret = previousSecret
		_ = sqlDB.Close()
	})
	user := &model.User{
		Username:    "sso-user",
		Password:    "unused-password-hash",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)
	now := time.Now().Unix()
	require.NoError(t, db.Create(&model.UserSession{
		SID:             "sso-session",
		UserID:          user.Id,
		Version:         1,
		UserAuthVersion: 1,
		Status:          model.UserSessionStatusActive,
		RefreshHash:     "refresh-hash",
		CreatedAt:       now,
		LastActiveAt:    now,
		ExpiresAt:       now + 300,
	}).Error)
	return user
}

func TestSSOAuthorizeAndTokenAreBoundAndSingleUse(t *testing.T) {
	user := setupSSOControllerTest(t)
	previousProjects := os.Getenv("AI_SSO_PROJECTS")
	os.Setenv("AI_SSO_PROJECTS", "hedezhongcheng|local-sso-client-secret|hdzc-shared|http://127.0.0.1:5175/sso/callback")
	t.Cleanup(func() { _ = os.Setenv("AI_SSO_PROJECTS", previousProjects) })

	authorizeResponse := httptest.NewRecorder()
	authorizeContext, _ := gin.CreateTestContext(authorizeResponse)
	authorizeContext.Request = httptest.NewRequest(http.MethodGet, "/api/sso/authorize?project_id=hedezhongcheng&redirect_uri=http%3A%2F%2F127.0.0.1%3A5175%2Fsso%2Fcallback&state=state-1", nil)
	authorizeContext.Set("id", user.Id)
	authorizeContext.Set("session_id", "sso-session")
	authorizeContext.Set("auth_version", int64(1))
	authorizeContext.Set("session_version", int64(1))

	SSOAuthorize(authorizeContext)
	require.Equal(t, http.StatusFound, authorizeResponse.Code)
	callback, err := url.Parse(authorizeResponse.Header().Get("Location"))
	require.NoError(t, err)
	code := callback.Query().Get("code")
	require.NotEmpty(t, code)
	assert.Equal(t, "state-1", callback.Query().Get("state"))

	tokenBody := `{"grant_type":"authorization_code","code":"` + code + `","project_id":"hedezhongcheng","redirect_uri":"http://127.0.0.1:5175/sso/callback"}`
	tokenResponse := httptest.NewRecorder()
	tokenContext, _ := gin.CreateTestContext(tokenResponse)
	tokenContext.Request = httptest.NewRequest(http.MethodPost, "/api/sso/token", strings.NewReader(tokenBody))
	tokenContext.Request.Header.Set("Authorization", "Bearer local-sso-client-secret")
	SSOToken(tokenContext)
	require.Equal(t, http.StatusOK, tokenResponse.Code)
	var tokenPayload struct {
		Data struct {
			Subject string `json:"subject"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(tokenResponse.Body.Bytes(), &tokenPayload))
	assert.Equal(t, strconv.Itoa(user.Id), tokenPayload.Data.Subject)

	replayResponse := httptest.NewRecorder()
	replayContext, _ := gin.CreateTestContext(replayResponse)
	replayContext.Request = httptest.NewRequest(http.MethodPost, "/api/sso/token", strings.NewReader(tokenBody))
	replayContext.Request.Header.Set("Authorization", "Bearer local-sso-client-secret")
	SSOToken(replayContext)
	assert.Equal(t, http.StatusBadRequest, replayResponse.Code)
}

func TestParseSSOProjectsRejectsUnsafeRedirect(t *testing.T) {
	_, err := parseSSOProjects("project|secret|enterprise|http://example.com/callback")
	assert.Error(t, err)

	projects, err := parseSSOProjects("project|secret|enterprise|http://127.0.0.1/callback\nproject|secret|enterprise|https://example.com/callback")
	require.NoError(t, err)
	assert.Len(t, projects["project"].RedirectURIs, 2)
}
