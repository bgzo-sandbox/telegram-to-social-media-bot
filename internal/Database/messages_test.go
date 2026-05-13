package Database

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"telegram-message-sync-bot/internal/Entity"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	err = db.AutoMigrate(&Entity.Message{}, &Entity.Attachment{}, &Entity.SyncRecord{})
	if err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	DB = db
}

func TestSaveMessage_DuplicateBySourceUniqueIndex(t *testing.T) {
	setupTestDB(t)

	msg1 := &Entity.Message{
		MessageID:   1001,
		Username:    "imbGZo",
		Content:     "first",
		MessageUrl:  "https://t.me/imbGZo/1001",
		MessageDate: time.Now(),
		CreatedTime: time.Now(),
	}

	_, err := SaveMessage(msg1)
	if err != nil {
		t.Fatalf("first save should succeed, got err: %v", err)
	}

	msg2 := &Entity.Message{
		MessageID:   1001,
		Username:    "imbGZo",
		Content:     "duplicate",
		MessageUrl:  "https://t.me/imbGZo/1001",
		MessageDate: time.Now(),
		CreatedTime: time.Now(),
	}

	_, err = SaveMessage(msg2)
	if err == nil {
		t.Fatalf("duplicate save should fail by unique constraint")
	}

	if !IsDuplicateMessageError(err) {
		t.Fatalf("expected duplicate error recognizer to return true, got err: %v", err)
	}
}

func TestSaveMessage_DifferentSourceCanCoexist(t *testing.T) {
	setupTestDB(t)

	msg1 := &Entity.Message{
		MessageID:   1001,
		Username:    "imbGZo",
		Content:     "first",
		MessageUrl:  "https://t.me/imbGZo/1001",
		MessageDate: time.Now(),
		CreatedTime: time.Now(),
	}

	msg2 := &Entity.Message{
		MessageID:   1001,
		Username:    "anotherChannel",
		Content:     "second",
		MessageUrl:  "https://t.me/anotherChannel/1001",
		MessageDate: time.Now(),
		CreatedTime: time.Now(),
	}

	if _, err := SaveMessage(msg1); err != nil {
		t.Fatalf("first save should succeed, got err: %v", err)
	}

	if _, err := SaveMessage(msg2); err != nil {
		t.Fatalf("second save with different source should succeed, got err: %v", err)
	}
}

func TestSaveSyncRecord_KeepsAttemptHistory(t *testing.T) {
	setupTestDB(t)

	msg := &Entity.Message{
		MessageID:   2001,
		Username:    "imbGZo",
		Content:     "hello",
		MessageUrl:  "https://t.me/imbGZo/2001",
		MessageDate: time.Now(),
		CreatedTime: time.Now(),
	}
	archivedMessageID, err := SaveMessage(msg)
	if err != nil {
		t.Fatalf("save message should succeed, got err: %v", err)
	}

	first := &Entity.SyncRecord{
		ArchivedMessageID: archivedMessageID,
		Platform:          "BlueSky",
		Status:            Entity.SyncStatusFailed,
		ErrorMessage:      "temporary failure",
		Trigger:           Entity.SyncTriggerAutomatic,
		CreatedTime:       time.Now(),
	}
	if _, err := SaveSyncRecord(first); err != nil {
		t.Fatalf("first sync record should save, got err: %v", err)
	}

	second := &Entity.SyncRecord{
		ArchivedMessageID: archivedMessageID,
		Platform:          "BlueSky",
		Status:            Entity.SyncStatusSucceeded,
		RemoteID:          "at://did:plc:test/app.bsky.feed.post/abc",
		Trigger:           Entity.SyncTriggerAutomatic,
		CreatedTime:       time.Now(),
	}
	if _, err := SaveSyncRecord(second); err != nil {
		t.Fatalf("second sync record should save, got err: %v", err)
	}

	records, err := ListSyncRecordsByMessage(archivedMessageID)
	if err != nil {
		t.Fatalf("list sync records should succeed, got err: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 sync records, got %d", len(records))
	}
	if records[0].AttemptNo != 1 || records[1].AttemptNo != 2 {
		t.Fatalf("unexpected attempt sequence: %+v", records)
	}

	latest, err := GetLatestSyncRecord(archivedMessageID, "BlueSky")
	if err != nil {
		t.Fatalf("get latest sync record should succeed, got err: %v", err)
	}
	if latest.Status != Entity.SyncStatusSucceeded {
		t.Fatalf("expected latest status succeeded, got: %+v", latest)
	}
	if latest.RemoteID == "" {
		t.Fatalf("expected latest record to keep remote id, got: %+v", latest)
	}
}
