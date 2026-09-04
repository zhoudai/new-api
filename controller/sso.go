package controller

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	ssoAuthorizationCodeTTL = time.Minute
	ssoStateMaxLength       = 512
)

type ssoProjectConfig struct {
	ProjectID    string
	ClientSecret string
	EnterpriseID string
	RedirectURIs map[string]struct{}
}

type ssoCodePayload struct {
	ProjectID    string `json:"project_id"`
	EnterpriseID string `json:"enterprise_id"`
	RedirectURI  string `json:"redirect_uri"`
}

type ssoTokenRequest struct {
	GrantType   string `json:"grant_type"`
	Code        string `json:"code"`
	ProjectID   string `json:"project_id"`
	RedirectURI string `json:"redirect_uri"`
}

// parseSSOProjects 从受控配置加载项目登记和企业绑定；正式管理界面可替换该配置源。
func parseSSOProjects(raw string) (map[string]ssoProjectConfig, error) {
	projects := make(map[string]ssoProjectConfig)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			return nil, errors.New("AI_SSO_PROJECTS 包含无效登记")
		}
		projectID := strings.TrimSpace(parts[0])
		clientSecret := strings.TrimSpace(parts[1])
		enterpriseID := strings.TrimSpace(parts[2])
		redirectURI := strings.TrimSpace(parts[3])
		if !validSSOIdentifier(projectID) || clientSecret == "" || len(clientSecret) > 256 ||
			!validSSOIdentifier(enterpriseID) || !validSSORedirectURI(redirectURI) {
			return nil, errors.New("AI_SSO_PROJECTS 包含无效登记")
		}

		project, exists := projects[projectID]
		if !exists {
			project = ssoProjectConfig{
				ProjectID:    projectID,
				ClientSecret: clientSecret,
				EnterpriseID: enterpriseID,
				RedirectURIs: make(map[string]struct{}),
			}
		} else if project.ClientSecret != clientSecret || project.EnterpriseID != enterpriseID {
			return nil, errors.New("AI_SSO_PROJECTS 同一项目登记不一致")
		}
		project.RedirectURIs[redirectURI] = struct{}{}
		projects[projectID] = project
	}
	if len(projects) == 0 {
		return nil, errors.New("AI_SSO_PROJECTS 未配置")
	}
	return projects, nil
}

func validSSOIdentifier(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for index, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			if index == 0 && !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
				return false
			}
			continue
		}
		return false
	}
	return true
}

func validSSORedirectURI(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Fragment != "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	return parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

func loadSSOProject(projectID string) (ssoProjectConfig, error) {
	projects, err := parseSSOProjects(os.Getenv("AI_SSO_PROJECTS"))
	if err != nil {
		return ssoProjectConfig{}, err
	}
	project, ok := projects[strings.TrimSpace(projectID)]
	if !ok {
		return ssoProjectConfig{}, fmt.Errorf("项目未登记")
	}
	return project, nil
}

func ssoBearerToken(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= len("Bearer ") || !strings.EqualFold(value[:len("Bearer ")], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(value[len("Bearer "):])
}

func ssoProjectAuthenticated(c *gin.Context, project ssoProjectConfig) bool {
	provided := ssoBearerToken(c.GetHeader("Authorization"))
	return provided != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(project.ClientSecret)) == 1
}

func ssoError(c *gin.Context, status int, code, message string) {
	c.Header("Cache-Control", "no-store")
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// SSOAuthorize 将当前平台登录会话转换为短时一次性授权码并回调业务项目。
func SSOAuthorize(c *gin.Context) {
	project, err := loadSSOProject(c.Query("project_id"))
	if err != nil {
		ssoError(c, http.StatusServiceUnavailable, "sso_not_configured", "SSO 项目未配置")
		return
	}
	redirectURI := strings.TrimSpace(c.Query("redirect_uri"))
	state := strings.TrimSpace(c.Query("state"))
	if _, ok := project.RedirectURIs[redirectURI]; !ok {
		ssoError(c, http.StatusBadRequest, "invalid_redirect_uri", "回调地址未登记")
		return
	}
	if state == "" || len(state) > ssoStateMaxLength {
		ssoError(c, http.StatusBadRequest, "invalid_state", "state 无效")
		return
	}
	identity, ok := middleware.GetSessionAuthIdentity(c)
	if !ok {
		ssoError(c, http.StatusUnauthorized, "login_required", "需要平台登录会话")
		return
	}
	user, err := model.GetUserById(identity.UserID, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		ssoError(c, http.StatusUnauthorized, "login_required", "平台登录会话无效")
		return
	}
	payload, err := common.Marshal(ssoCodePayload{
		ProjectID:    project.ProjectID,
		EnterpriseID: project.EnterpriseID,
		RedirectURI:  redirectURI,
	})
	if err != nil {
		ssoError(c, http.StatusInternalServerError, "sso_internal_error", "SSO 暂时不可用")
		return
	}
	code, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeSSO,
		Provider:  project.ProjectID,
		UserId:    identity.UserID,
		SessionId: identity.SessionID,
		Payload:   string(payload),
		ExpiresAt: time.Now().Add(ssoAuthorizationCodeTTL),
	})
	if err != nil {
		ssoError(c, http.StatusInternalServerError, "sso_internal_error", "SSO 暂时不可用")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, mustAppendSSOQuery(redirectURI, code, state))
}

func mustAppendSSOQuery(rawURI, code, state string) string {
	parsed, _ := url.Parse(rawURI)
	query := parsed.Query()
	query.Set("code", code)
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// SSOToken 用项目密钥兑换授权码，仅返回业务项目建立本地会话所需的最小身份。
func SSOToken(c *gin.Context) {
	var request ssoTokenRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.GrantType != "authorization_code" {
		ssoError(c, http.StatusBadRequest, "invalid_grant", "授权请求无效")
		return
	}
	project, err := loadSSOProject(request.ProjectID)
	if err != nil {
		ssoError(c, http.StatusUnauthorized, "invalid_client", "项目未登记")
		return
	}
	if !ssoProjectAuthenticated(c, project) {
		ssoError(c, http.StatusUnauthorized, "invalid_client", "项目凭证无效")
		return
	}
	redirectURI := strings.TrimSpace(request.RedirectURI)
	if request.Code == "" {
		ssoError(c, http.StatusBadRequest, "invalid_grant", "授权码无效")
		return
	}
	flow, err := model.GetAuthFlow(request.Code, model.AuthFlowMatch{
		Purpose:  model.AuthFlowPurposeSSO,
		Provider: project.ProjectID,
	})
	if err != nil || flow.SessionId == "" {
		ssoError(c, http.StatusBadRequest, "invalid_grant", "授权码无效或已过期")
		return
	}
	if _, err := service.ValidateSessionReference(flow.UserId, flow.SessionId); err != nil {
		ssoError(c, http.StatusUnauthorized, "login_required", "平台登录会话已失效")
		return
	}
	var payload ssoCodePayload
	if err := common.UnmarshalJsonStr(flow.Payload, &payload); err != nil ||
		payload.ProjectID != project.ProjectID || payload.EnterpriseID != project.EnterpriseID || payload.RedirectURI != redirectURI {
		ssoError(c, http.StatusBadRequest, "invalid_grant", "授权码绑定不匹配")
		return
	}
	if _, err := model.ConsumeAuthFlow(request.Code, model.AuthFlowMatch{
		Purpose:   model.AuthFlowPurposeSSO,
		Provider:  project.ProjectID,
		UserId:    flow.UserId,
		SessionId: flow.SessionId,
	}); err != nil {
		ssoError(c, http.StatusBadRequest, "invalid_grant", "授权码无效或已使用")
		return
	}
	user, err := model.GetUserById(flow.UserId, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		ssoError(c, http.StatusUnauthorized, "login_required", "平台用户无效")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"project_id":    project.ProjectID,
			"enterprise_id": project.EnterpriseID,
			"subject":       strconv.Itoa(user.Id),
		},
	})
}
