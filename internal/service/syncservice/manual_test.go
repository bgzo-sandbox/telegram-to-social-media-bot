package syncservice

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
)

type manualFakeSender struct {
	name   string
	result DispatchResult
	called bool
}

func (f *manualFakeSender) Name() string {
	return f.name
}

func (f *manualFakeSender) Send(_ Entity.Config, payload Payload) DispatchResult {
	f.called = true
	f.result.ImageRequested = payload.Image != nil
	f.result.UsedImage = payload.Image != nil && f.result.Success
	if f.result.Platform == "" {
		f.result.Platform = f.name
	}
	return f.result
}

func setupManualSyncTestDB(t *testing.T) int64 {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory db: %v", err)
	}
	if err := db.AutoMigrate(&Entity.Message{}, &Entity.Attachment{}, &Entity.SyncRecord{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}
	Database.DB = db

	messageID, err := Database.SaveMessage(&Entity.Message{
		MessageID:   4001,
		Username:    "imbGZo",
		Content:     "hello",
		MessageUrl:  "https://t.me/imbGZo/4001",
		MessageDate: time.Now(),
		Attachments: []Entity.Attachment{{FilePath: "/tmp/test.jpg", Type: Entity.ImageMessage}},
		CreatedTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to save message: %v", err)
	}
	return messageID
}

func TestManualResync_SkipWhenSourceNotTarget(t *testing.T) {
	archivedMessageID := setupManualSyncTestDB(t)

	config := Entity.Config{}
	config.SocialMediaSync.TargetChannel = []string{"other"}

	result, err := ManualResync(config, archivedMessageID, "all")
	if err != nil {
		t.Fatalf("manual resync should not error, got: %v", err)
	}
	if result.Requested {
		t.Fatalf("expected manual resync to be skipped: %+v", result)
	}
}

func TestManualResync_SinglePlatformAndPersistManualTrigger(t *testing.T) {
	archivedMessageID := setupManualSyncTestDB(t)

	sender := &manualFakeSender{name: "Twitter", result: DispatchResult{Success: true, RemoteID: "123", RemoteURL: "https://twitter.com/i/web/status/123"}}
	originalFactory := manualDispatchFactory
	manualDispatchFactory = func() []Sender {
		return []Sender{sender}
	}
	defer func() {
		manualDispatchFactory = originalFactory
	}()

	config := Entity.Config{}
	config.SocialMediaSync.TargetChannel = []string{"imbGZo"}

	result, err := ManualResync(config, archivedMessageID, "twitter")
	if err != nil {
		t.Fatalf("manual resync failed: %v", err)
	}
	if !result.Requested || len(result.Results) != 1 {
		t.Fatalf("unexpected manual resync result: %+v", result)
	}
	if !sender.called {
		t.Fatalf("expected manual sender to be called")
	}

	records, err := Database.ListSyncRecordsByMessage(archivedMessageID)
	if err != nil {
		t.Fatalf("failed to load sync records: %v", err)
	}
	if len(records) != 1 || records[0].Trigger != Entity.SyncTriggerManual {
		t.Fatalf("unexpected persisted manual sync record: %+v", records)
	}
}

func TestManualResync_UnknownPlatform(t *testing.T) {
	archivedMessageID := setupManualSyncTestDB(t)

	config := Entity.Config{}
	config.SocialMediaSync.TargetChannel = []string{"imbGZo"}

	_, err := ManualResync(config, archivedMessageID, "unknown")
	if err == nil {
		t.Fatalf("expected unknown platform error")
	}
}
