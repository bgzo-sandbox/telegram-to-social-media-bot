package Database

import (
	"fmt"
	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/pkg/LogUtils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitORMDB(dataDir string) error {
	dbPath := dataDir + "/archive.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open gorm sqlite db: %w", err)
	}
	// 自动迁移表结构
	err = db.AutoMigrate(&Entity.Message{}, &Entity.Attachment{}, &Entity.SyncRecord{})
	if err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	DB = db
	LogUtils.GetLogger().Println("Database initialized.")
	return nil
}
