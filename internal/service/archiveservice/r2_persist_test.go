package archiveservice

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
)

// setupPersistTest 准备离线 PersistMessage 环境：伪 Telegram API、临时模板与数据库。
func setupPersistTest(t *testing.T) (Entity.Config, *bot.Bot, string) {
	t.Helper()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/getFile") {
			w.Write([]byte(`{"ok":true,"result":{"file_id":"FILE1","file_unique_id":"UNIQ1","file_path":"photos/photo.jpg"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(apiServer.Close)

	b, err := bot.New("test-token", bot.WithServerURL(apiServer.URL), bot.WithSkipGetMe())
	if err != nil {
		t.Fatalf("创建 bot 失败: %v", err)
	}

	oldDownload := downloadFile
	downloadFile = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("imgdata")),
		}, nil
	}
	t.Cleanup(func() { downloadFile = oldDownload })

	root := t.TempDir()
	tmplPath := filepath.Join(root, "template.txt")
	if err := os.WriteFile(tmplPath, []byte("{{.photo}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(root, "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Entity.Message{}, &Entity.Attachment{}, &Entity.SyncRecord{}); err != nil {
		t.Fatal(err)
	}
	oldDB := Database.DB
	Database.DB = db
	t.Cleanup(func() { Database.DB = oldDB })

	cfg := Entity.Config{}
	cfg.Output.PersonDir = filepath.Join(root, "person")
	cfg.Output.ChannelDir = filepath.Join(root, "channel")
	cfg.Template.Dir = tmplPath

	return cfg, b, root
}

func photoUpdate() *models.Update {
	return &models.Update{
		Message: &models.Message{
			ID:   99,
			Chat: models.Chat{ID: 12345},
			Photo: []models.PhotoSize{
				{FileID: "FILE0", FileUniqueID: "UNIQ0"},
				{FileID: "FILE1", FileUniqueID: "UNIQ1"},
			},
		},
	}
}

func TestPersistMessage_R2Disabled_KeepsLocalPath(t *testing.T) {
	cfg, b, _ := setupPersistTest(t)
	cfg.R2.Enable = false

	result := PersistMessage(context.Background(), b, photoUpdate(), cfg)
	if !result.OK {
		t.Fatalf("归档应成功: %s", result.Message)
	}

	markdownPath := filepath.Join(cfg.Output.PersonDir, "12345", "99.md")
	content, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("读取归档文件失败: %v", err)
	}
	if !strings.Contains(string(content), "assets/12345/UNIQ1") {
		t.Fatalf("R2 关闭时 Markdown 应使用本地路径:\n%s", content)
	}

	msg, err := Database.GetMessageBySource(99, "12345")
	if err != nil {
		t.Fatalf("读取入库消息失败: %v", err)
	}
	if len(msg.Attachments) != 1 {
		t.Fatalf("应有 1 个附件, got %d", len(msg.Attachments))
	}
	if msg.Attachments[0].S3Url != "" {
		t.Fatalf("R2 关闭时 S3Url 应为空, got %q", msg.Attachments[0].S3Url)
	}
}

func TestPersistMessage_R2Enabled_FillsS3URL(t *testing.T) {
	cfg, b, _ := setupPersistTest(t)
	cfg.R2.Enable = true
	cfg.R2.Path = "tg-archive"

	fakeURL := "https://media.example.com/tg-archive/12345/UNIQ1"
	oldFactory := uploadAttachmentFactory
	uploadAttachmentFactory = func(_ context.Context, _ Entity.Config, meta SourceMeta, file *Entity.Attachment) (string, error) {
		if meta.SourceID != "12345" || meta.MessageID != 99 {
			t.Fatalf("meta 不匹配: %+v", meta)
		}
		if file.Type != Entity.ImageMessage {
			t.Fatalf("仅 image 应触发上传, got %s", file.Type)
		}
		return fakeURL, nil
	}
	t.Cleanup(func() { uploadAttachmentFactory = oldFactory })

	result := PersistMessage(context.Background(), b, photoUpdate(), cfg)
	if !result.OK {
		t.Fatalf("归档应成功: %s", result.Message)
	}

	markdownPath := filepath.Join(cfg.Output.PersonDir, "12345", "99.md")
	content, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("读取归档文件失败: %v", err)
	}
	if !strings.Contains(string(content), fakeURL) {
		t.Fatalf("R2 开启时 Markdown 应使用 R2 URL:\n%s", content)
	}
	if strings.Contains(string(content), "assets/12345/UNIQ1") {
		t.Fatalf("R2 开启时 Markdown 不应使用本地路径:\n%s", content)
	}

	if result.ImagePath != "assets/12345/UNIQ1" {
		t.Fatalf("ImagePath 应保持本地相对路径, got %q", result.ImagePath)
	}

	msg, err := Database.GetMessageBySource(99, "12345")
	if err != nil {
		t.Fatalf("读取入库消息失败: %v", err)
	}
	if msg.Attachments[0].S3Url != fakeURL {
		t.Fatalf("S3Url 应写入 fake URL, got %q", msg.Attachments[0].S3Url)
	}
}

func TestPersistMessage_R2UploadFailed_KeepsLocalPath(t *testing.T) {
	cfg, b, _ := setupPersistTest(t)
	cfg.R2.Enable = true

	oldFactory := uploadAttachmentFactory
	uploadAttachmentFactory = func(_ context.Context, _ Entity.Config, _ SourceMeta, _ *Entity.Attachment) (string, error) {
		return "", errors.New("boom")
	}
	t.Cleanup(func() { uploadAttachmentFactory = oldFactory })

	result := PersistMessage(context.Background(), b, photoUpdate(), cfg)
	if !result.OK {
		t.Fatalf("R2 上传失败不应中断归档: %s", result.Message)
	}

	markdownPath := filepath.Join(cfg.Output.PersonDir, "12345", "99.md")
	content, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("读取归档文件失败: %v", err)
	}
	if !strings.Contains(string(content), "assets/12345/UNIQ1") {
		t.Fatalf("上传失败时 Markdown 应回落本地路径:\n%s", content)
	}

	msg, err := Database.GetMessageBySource(99, "12345")
	if err != nil {
		t.Fatalf("读取入库消息失败: %v", err)
	}
	if msg.Attachments[0].S3Url != "" {
		t.Fatalf("上传失败时 S3Url 应保持空, got %q", msg.Attachments[0].S3Url)
	}
}
