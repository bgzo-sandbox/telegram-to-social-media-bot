package jsonbackfillservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
)

func setupBackfillTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory db: %v", err)
	}
	if err := db.AutoMigrate(&Entity.Message{}, &Entity.Attachment{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}
	Database.DB = db
}

func writeUpdateJSON(t *testing.T, path string, update models.Update, modTime time.Time) {
	t.Helper()

	data, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("failed to marshal update: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create json dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write json file: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("failed to set file times: %v", err)
	}
}

func testConfig(jsonDir string) Entity.Config {
	var cfg Entity.Config
	cfg.Output.JsonDir = jsonDir
	cfg.Output.PersonDir = filepath.Join(jsonDir, "..", "person")
	cfg.Output.ChannelDir = filepath.Join(jsonDir, "..", "channel")
	return cfg
}

func TestBackfillFromJSON_ImportsPrivateMessage(t *testing.T) {
	setupBackfillTestDB(t)
	tmp := t.TempDir()
	jsonRoot := filepath.Join(tmp, "json")
	filePath := filepath.Join(jsonRoot, "20250329", "private.json")
	modTime := time.Date(2026, 5, 4, 9, 30, 0, 0, time.UTC)

	writeUpdateJSON(t, filePath, models.Update{
		Message: &models.Message{
			ID:   658,
			Date: 1743233467,
			Chat: models.Chat{ID: 845458984, Type: "private"},
			Text: "你好",
		},
	}, modTime)

	stats, err := BackfillFromJSON(testConfig(jsonRoot))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if stats.Scanned != 1 || stats.Decoded != 1 || stats.Inserted != 1 || stats.Duplicates != 0 || stats.Skipped != 0 || stats.Failed != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	msg, err := Database.GetMessageBySource(658, "845458984")
	if err != nil {
		t.Fatalf("expected saved private message, got err: %v", err)
	}
	if msg.Content != "你好" {
		t.Fatalf("unexpected content: %s", msg.Content)
	}
	if !msg.MessageDate.Equal(time.Unix(1743233467, 0)) {
		t.Fatalf("unexpected message date: %v", msg.MessageDate)
	}
	if !msg.CreatedTime.Equal(modTime) {
		t.Fatalf("unexpected created time: %v", msg.CreatedTime)
	}
}

func TestBackfillFromJSON_ImportsForwardedChannelMessage(t *testing.T) {
	setupBackfillTestDB(t)
	tmp := t.TempDir()
	jsonRoot := filepath.Join(tmp, "json")
	filePath := filepath.Join(jsonRoot, "20250330", "channel.json")

	writeUpdateJSON(t, filePath, models.Update{
		Message: &models.Message{
			ID:   1,
			Date: 1743233467,
			Chat: models.Chat{ID: 845458984, Type: "private"},
			Text: "频道消息",
			ForwardOrigin: &models.MessageOrigin{
				Type: "channel",
				MessageOriginChannel: &models.MessageOriginChannel{
					Date:      1700000000,
					MessageID: 321,
					Chat:      models.Chat{ID: -1001, Username: "imbGZo"},
				},
			},
		},
	}, time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC))

	stats, err := BackfillFromJSON(testConfig(jsonRoot))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if stats.Inserted != 1 {
		t.Fatalf("expected one insert, got stats: %+v", stats)
	}

	msg, err := Database.GetMessageBySource(321, "imbGZo")
	if err != nil {
		t.Fatalf("expected saved channel message, got err: %v", err)
	}
	if msg.MessageUrl != "https://t.me/imbGZo/321" {
		t.Fatalf("unexpected message url: %s", msg.MessageUrl)
	}
	if !msg.MessageDate.Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("unexpected message date: %v", msg.MessageDate)
	}
}

func TestBackfillFromJSON_DuplicateMessagesAreCounted(t *testing.T) {
	setupBackfillTestDB(t)
	tmp := t.TempDir()
	jsonRoot := filepath.Join(tmp, "json")
	filePath := filepath.Join(jsonRoot, "20250331", "duplicate.json")

	writeUpdateJSON(t, filePath, models.Update{
		Message: &models.Message{
			ID:   777,
			Date: 1743233467,
			Chat: models.Chat{ID: 845458984, Type: "private"},
			Text: "重复消息",
		},
	}, time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC))

	first, err := BackfillFromJSON(testConfig(jsonRoot))
	if err != nil {
		t.Fatalf("expected first import success, got: %v", err)
	}
	if first.Inserted != 1 {
		t.Fatalf("expected first insert, got stats: %+v", first)
	}

	second, err := BackfillFromJSON(testConfig(jsonRoot))
	if err != nil {
		t.Fatalf("expected duplicate import to be non-fatal, got: %v", err)
	}
	if second.Inserted != 0 || second.Duplicates != 1 {
		t.Fatalf("expected duplicate stats, got: %+v", second)
	}
}

func TestBackfillFromJSON_InvalidJSONDoesNotBlockValidFiles(t *testing.T) {
	setupBackfillTestDB(t)
	tmp := t.TempDir()
	jsonRoot := filepath.Join(tmp, "json")
	validPath := filepath.Join(jsonRoot, "20250401", "valid.json")
	invalidPath := filepath.Join(jsonRoot, "20250401", "invalid.json")

	writeUpdateJSON(t, validPath, models.Update{
		Message: &models.Message{
			ID:   888,
			Date: 1743233467,
			Chat: models.Chat{ID: 845458984, Type: "private"},
			Text: "合法消息",
		},
	}, time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC))
	if err := os.MkdirAll(filepath.Dir(invalidPath), 0o755); err != nil {
		t.Fatalf("failed to create invalid dir: %v", err)
	}
	if err := os.WriteFile(invalidPath, []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("failed to write invalid json: %v", err)
	}

	stats, err := BackfillFromJSON(testConfig(jsonRoot))
	if err == nil {
		t.Fatalf("expected aggregated error when invalid json exists")
	}
	if stats.Inserted != 1 || stats.Failed != 1 || stats.Scanned != 2 || stats.Decoded != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if !strings.Contains(err.Error(), "invalid.json") {
		t.Fatalf("expected aggregated error to mention invalid file, got: %v", err)
	}

	msg, err := Database.GetMessageBySource(888, "845458984")
	if err != nil {
		t.Fatalf("expected valid message to be inserted, got err: %v", err)
	}
	if msg.Content != "合法消息" {
		t.Fatalf("unexpected content: %s", msg.Content)
	}
}
