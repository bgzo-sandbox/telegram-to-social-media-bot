package adminqueryservice

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
)

func setupAdminQueryTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory db: %v", err)
	}
	if err := db.AutoMigrate(&Entity.Message{}, &Entity.Attachment{}, &Entity.SyncRecord{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}
	Database.DB = db
}

func seedAdminQueryData(t *testing.T) (int64, int64) {
	t.Helper()

	now := time.Now()
	msg1, err := Database.SaveMessage(&Entity.Message{
		MessageID:   101,
		Username:    "imbGZo",
		Content:     "hello one",
		MessageUrl:  "https://t.me/imbGZo/101",
		MessageDate: now,
		CreatedTime: now,
	})
	if err != nil {
		t.Fatalf("failed to save msg1: %v", err)
	}

	msg2, err := Database.SaveMessage(&Entity.Message{
		MessageID:   102,
		Username:    "imbGZo",
		Content:     "hello two",
		MessageUrl:  "https://t.me/imbGZo/102",
		MessageDate: now.Add(time.Minute),
		CreatedTime: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("failed to save msg2: %v", err)
	}

	other, err := Database.SaveMessage(&Entity.Message{
		MessageID:   201,
		Username:    "other",
		Content:     "other content",
		MessageUrl:  "https://t.me/other/201",
		MessageDate: now,
		CreatedTime: now,
	})
	if err != nil {
		t.Fatalf("failed to save other msg: %v", err)
	}

	_, _ = Database.SaveSyncRecord(&Entity.SyncRecord{
		ArchivedMessageID: msg1,
		Platform:          "BlueSky",
		Status:            Entity.SyncStatusFailed,
		ErrorMessage:      "first fail",
		Trigger:           Entity.SyncTriggerAutomatic,
		CreatedTime:       now,
	})
	_, _ = Database.SaveSyncRecord(&Entity.SyncRecord{
		ArchivedMessageID: msg1,
		Platform:          "BlueSky",
		Status:            Entity.SyncStatusSucceeded,
		RemoteID:          "at://did:plc:test/app.bsky.feed.post/xyz",
		Trigger:           Entity.SyncTriggerAutomatic,
		CreatedTime:       now.Add(time.Minute),
	})
	_, _ = Database.SaveSyncRecord(&Entity.SyncRecord{
		ArchivedMessageID: msg2,
		Platform:          "Twitter",
		Status:            Entity.SyncStatusFailed,
		ErrorMessage:      "too long",
		Trigger:           Entity.SyncTriggerAutomatic,
		CreatedTime:       now.Add(2 * time.Minute),
	})
	_, _ = Database.SaveSyncRecord(&Entity.SyncRecord{
		ArchivedMessageID: other,
		Platform:          "Mastodon",
		Status:            Entity.SyncStatusSucceeded,
		RemoteURL:         "https://mastodon.example/@user/1",
		Trigger:           Entity.SyncTriggerAutomatic,
		CreatedTime:       now,
	})

	return msg1, msg2
}

func TestLoadOverview(t *testing.T) {
	setupAdminQueryTestDB(t)
	seedAdminQueryData(t)

	config := Entity.Config{}
	config.SocialMediaSync.TargetChannel = []string{"imbGZo", "chan2"}

	overview, err := LoadOverview(config)
	if err != nil {
		t.Fatalf("load overview failed: %v", err)
	}
	if overview.MessageCount != 3 {
		t.Fatalf("unexpected message count: %+v", overview)
	}
	if overview.SourceCount != 2 {
		t.Fatalf("unexpected source count: %+v", overview)
	}
	if overview.SyncTargetSourceCount != 2 {
		t.Fatalf("unexpected sync target count: %+v", overview)
	}
}

func TestListSources(t *testing.T) {
	setupAdminQueryTestDB(t)
	seedAdminQueryData(t)

	config := Entity.Config{}
	config.SocialMediaSync.TargetChannel = []string{"imbGZo"}

	sources, err := ListSources(config, false)
	if err != nil {
		t.Fatalf("list sources failed: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].SourceID != "imbGZo" || !sources[0].SyncTarget {
		t.Fatalf("unexpected first source: %+v", sources[0])
	}

	targetsOnly, err := ListSources(config, true)
	if err != nil {
		t.Fatalf("list target sources failed: %v", err)
	}
	if len(targetsOnly) != 1 || targetsOnly[0].SourceID != "imbGZo" {
		t.Fatalf("unexpected target-only sources: %+v", targetsOnly)
	}
}

func TestListMessagesBySource(t *testing.T) {
	setupAdminQueryTestDB(t)
	_, latestMessageID := seedAdminQueryData(t)

	messages, err := ListMessagesBySource("imbGZo")
	if err != nil {
		t.Fatalf("list messages by source failed: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].ArchivedMessageID != latestMessageID {
		t.Fatalf("expected latest message first, got: %+v", messages[0])
	}
	if !messages[0].HasSyncFailure {
		t.Fatalf("expected latest message to show sync failure: %+v", messages[0])
	}
}

func TestListFailedSyncMessages(t *testing.T) {
	setupAdminQueryTestDB(t)
	_, failedMessageID := seedAdminQueryData(t)

	config := Entity.Config{}
	config.SocialMediaSync.TargetChannel = []string{"imbGZo"}

	messages, err := ListFailedSyncMessages(config)
	if err != nil {
		t.Fatalf("list failed sync messages failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 failed sync message, got %d", len(messages))
	}
	if messages[0].ArchivedMessageID != failedMessageID {
		t.Fatalf("unexpected failed sync message: %+v", messages[0])
	}
}

func TestGetMessageDetail(t *testing.T) {
	setupAdminQueryTestDB(t)
	messageID, _ := seedAdminQueryData(t)

	detail, err := GetMessageDetail(messageID)
	if err != nil {
		t.Fatalf("get message detail failed: %v", err)
	}
	if detail.SourceID != "imbGZo" {
		t.Fatalf("unexpected detail source: %+v", detail)
	}
	if len(detail.LatestStatuses) != 1 {
		t.Fatalf("expected one latest status, got %+v", detail.LatestStatuses)
	}
	if detail.LatestStatuses[0].Status != string(Entity.SyncStatusSucceeded) {
		t.Fatalf("expected latest status succeeded, got %+v", detail.LatestStatuses[0])
	}
	if detail.LatestStatuses[0].RemoteID == "" {
		t.Fatalf("expected remote id in detail status, got %+v", detail.LatestStatuses[0])
	}
}
