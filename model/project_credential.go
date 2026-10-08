package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrProjectCredentialConflict = errors.New("项目凭证不存在或版本已变化")
var projectCredentialIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.:-]{0,63}$`)

// ProjectCredential 仅保存随机凭证的摘要；明文只在生成和轮换响应中返回。
type ProjectCredential struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	ProjectID  string    `json:"project_id" gorm:"type:varchar(64);not null;uniqueIndex"`
	SecretHash string    `json:"-" gorm:"type:char(64);not null"`
	Version    int64     `json:"version" gorm:"not null"`
	Status     string    `json:"status" gorm:"type:varchar(16);not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// 审计与凭证在主库同一事务写入，避免独立日志库故障造成无审计的变更。
type ProjectCredentialAudit struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	CredentialID int64     `json:"credential_id" gorm:"index;not null"`
	ActorID      int       `json:"actor_id" gorm:"not null"`
	Action       string    `json:"action" gorm:"type:varchar(16);not null"`
	Version      int64     `json:"version" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
}

func ValidProjectCredentialID(id string) bool { return projectCredentialIDPattern.MatchString(id) }

func ChangeProjectCredential(projectID string, version int64, action string, actorID int) (*ProjectCredential, string, error) {
	if !ValidProjectCredentialID(projectID) || actorID <= 0 || (action != "create" && action != "rotate" && action != "disable") || (action != "create" && (version < 1 || version >= 2147483647)) {
		return nil, "", errors.New("项目凭证参数无效")
	}
	secret, hash := "", ""
	if action != "disable" {
		var random [32]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, "", err
		}
		secret = "apc_" + hex.EncodeToString(random[:])
		digest := sha256.Sum256([]byte(secret))
		hash = hex.EncodeToString(digest[:])
	}
	credential := ProjectCredential{ProjectID: projectID, Version: 1, Status: "active", SecretHash: hash}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if action == "create" {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&credential)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrProjectCredentialConflict
			}
		} else {
			status := "active"
			if action == "disable" {
				status = "disabled"
			}
			// 版本条件让并发轮换/禁用只能成功一次；禁用也清除旧摘要。
			result := tx.Model(&ProjectCredential{}).Where("project_id = ? AND version = ?", projectID, version).
				Updates(map[string]interface{}{"secret_hash": hash, "status": status, "version": version + 1})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrProjectCredentialConflict
			}
			if err := tx.Where("project_id = ?", projectID).First(&credential).Error; err != nil {
				return err
			}
		}
		return tx.Create(&ProjectCredentialAudit{CredentialID: credential.ID, ActorID: actorID, Action: action, Version: credential.Version}).Error
	})
	if err != nil {
		return nil, "", err
	}
	return &credential, secret, nil
}

func VerifyProjectCredential(projectID, secret string) (bool, error) {
	if !ValidProjectCredentialID(projectID) || len(secret) != 68 || secret[:4] != "apc_" {
		return false, nil
	}
	digest := sha256.Sum256([]byte(secret))
	var count int64
	err := DB.Model(&ProjectCredential{}).Where("project_id = ? AND secret_hash = ? AND status = ?", projectID, hex.EncodeToString(digest[:]), "active").Count(&count).Error
	return count == 1, err
}
