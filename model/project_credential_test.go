package model

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// 三种数据库使用隔离表前缀，保留测试数据供复核，不触及已有业务表。
func TestProjectCredentialLifecycle(t *testing.T) {
	for _, engine := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			var dialect gorm.Dialector = sqlite.Open(":memory:")
			if engine == "mysql" {
				if os.Getenv("TEST_MYSQL_DSN") == "" {
					t.Skip("TEST_MYSQL_DSN 未配置")
				}
				dialect = mysql.Open(os.Getenv("TEST_MYSQL_DSN"))
			}
			if engine == "postgres" {
				if os.Getenv("TEST_POSTGRES_DSN") == "" {
					t.Skip("TEST_POSTGRES_DSN 未配置")
				}
				dialect = postgres.Open(os.Getenv("TEST_POSTGRES_DSN"))
			}
			db, err := gorm.Open(dialect, &gorm.Config{NamingStrategy: schema.NamingStrategy{TablePrefix: fmt.Sprintf("pc%d_", time.Now().UnixNano())}})
			require.NoError(t, err)
			previous := DB
			if engine == "sqlite" {
				var version string
				require.NoError(t, db.Raw("SELECT sqlite_version()").Scan(&version).Error)
				t.Log("SQLite", version)
			}
			DB = db
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { DB = previous; require.NoError(t, sqlDB.Close()) })
			require.NoError(t, db.AutoMigrate(&ProjectCredential{}, &ProjectCredentialAudit{}))
			credential, secret, err := ChangeProjectCredential("project-a", 0, "create", 1)
			require.NoError(t, err)
			require.NotEmpty(t, secret)
			// 重复启动迁移不改变现有凭证、审计和唯一性。
			for range 2 {
				require.NoError(t, db.AutoMigrate(&ProjectCredential{}, &ProjectCredentialAudit{}))
			}
			ok, err := VerifyProjectCredential("project-a", secret)
			require.NoError(t, err)
			assert.True(t, ok)
			ok, err = VerifyProjectCredential("project-b", secret)
			require.NoError(t, err)
			assert.False(t, ok)
			_, _, err = ChangeProjectCredential("project-a", 0, "create", 1)
			require.Error(t, err)
			encoded, err := common.Marshal(credential)
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), secret)
			assert.NotContains(t, string(encoded), credential.SecretHash)
			rotated, next, err := ChangeProjectCredential("project-a", credential.Version, "rotate", 2)
			require.NoError(t, err)
			_, _, err = ChangeProjectCredential("project-a", credential.Version, "disable", 2)
			require.ErrorIs(t, err, ErrProjectCredentialConflict)
			ok, err = VerifyProjectCredential("project-a", secret)
			require.NoError(t, err)
			assert.False(t, ok)
			ok, err = VerifyProjectCredential("project-a", next)
			require.NoError(t, err)
			assert.True(t, ok)
			_, returned, err := ChangeProjectCredential("project-a", rotated.Version, "disable", 2)
			require.NoError(t, err)
			assert.Empty(t, returned)
			ok, err = VerifyProjectCredential("project-a", next)
			require.NoError(t, err)
			assert.False(t, ok)
			var audits []ProjectCredentialAudit
			require.NoError(t, db.Order("id").Find(&audits).Error)
			require.Len(t, audits, 3)
			assert.Equal(t, "create", audits[0].Action)
			assert.Equal(t, "rotate", audits[1].Action)
			assert.Equal(t, "disable", audits[2].Action)
			assert.Equal(t, 2, audits[2].ActorID)
			// 审计写入失败时，凭证变更必须回滚。
			require.NoError(t, db.Callback().Create().Before("gorm:create").Register("reject_credential_audit", func(tx *gorm.DB) {
				if _, ok := tx.Statement.Model.(*ProjectCredentialAudit); ok {
					tx.AddError(fmt.Errorf("测试审计故障"))
				}
			}))
			_, returned, err = ChangeProjectCredential("project-a", 3, "rotate", 2)
			require.Error(t, err)
			assert.Empty(t, returned)
			var stored ProjectCredential
			require.NoError(t, db.First(&stored, credential.ID).Error)
			assert.Equal(t, "disabled", stored.Status)
			assert.EqualValues(t, 3, stored.Version)
		})
	}
}
