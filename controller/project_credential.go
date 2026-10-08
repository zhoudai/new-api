package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func ListProjectCredentials(c *gin.Context) {
	// 有界游标分页，状态查询永不返回明文或摘要。
	after, err := strconv.ParseInt(c.DefaultQuery("after", "0"), 10, 64)
	if err != nil || after < 0 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	items := []model.ProjectCredential{}
	if err := model.DB.Where("id > ?", after).Order("id ASC").Limit(100).Find(&items).Error; err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func ChangeProjectCredential(c *gin.Context) {
	var request struct {
		ProjectID string `json:"project_id"`
		Version   int64  `json:"version"`
	}
	if common.DecodeJson(c.Request.Body, &request) != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	action := c.Param("action")
	if action == "" {
		action = "create"
	} else {
		request.ProjectID = c.Param("project_id")
	}
	if !model.ValidProjectCredentialID(request.ProjectID) || (action != "create" && action != "rotate" && action != "disable") || (action != "create" && (request.Version < 1 || request.Version >= 2147483647)) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	credential, secret, err := model.ChangeProjectCredential(request.ProjectID, request.Version, action, c.GetInt("id"))
	if errors.Is(err, model.ErrProjectCredentialConflict) {
		c.AbortWithStatus(http.StatusConflict)
		return
	}
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	// no-store 由路由统一设置；响应丢失后只能轮换，不能再次读取凭证。
	data := gin.H{"credential": credential}
	if secret != "" {
		data["secret"] = secret
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func ListProjectCredentialAudits(c *gin.Context) {
	after, err := strconv.ParseInt(c.DefaultQuery("after", "0"), 10, 64)
	if err != nil || after < 0 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	items := []model.ProjectCredentialAudit{}
	if err := model.DB.Where("id > ?", after).Order("id ASC").Limit(100).Find(&items).Error; err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// VerifyProjectCredential 仅返回校验结果，不向调用方暴露内部计费 Key。
func VerifyProjectCredential(c *gin.Context) {
	ok, err := model.VerifyProjectCredential(c.GetHeader("X-AI-Project-Id"), ssoBearerToken(c.GetHeader("Authorization")))
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.Status(http.StatusNoContent)
}
