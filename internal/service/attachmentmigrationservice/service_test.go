package attachmentmigrationservice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/pkg/S3Utils"
)

type fakeR2Uploader struct {
	keys []string
	err  error
}

func (f *fakeR2Uploader) Upload(_ context.Context, _ string, key string) (string, error) {
	f.keys = append(f.keys, key)
	if f.err != nil {
		return "", f.err
	}
	return "https://media.example.com/" + key, nil
}

type migrationTestEnv struct {
	cfg        Entity.Config
	root       string
	fake       *fakeR2Uploader
	oldFactory func(Entity.Config) (S3Utils.Uploader, error)
}

func setupMigrationTest(t *testing.T) *migrationTestEnv {
	t.Helper()

	root := t.TempDir()
	channelDir := filepath.Join(root, "channel")
	personDir := filepath.Join(root, "person")
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
	cfg.R2.Enable = true
	cfg.R2.Path = "tg-archive"
	cfg.Output.ChannelDir = channelDir
	cfg.Output.PersonDir = personDir
	cfg.Template.Dir = tmplPath

	fake := &fakeR2Uploader{}
	oldFactory := attachmentMigrationUploaderFactory
	attachmentMigrationUploaderFactory = func(Entity.Config) (S3Utils.Uploader, error) {
		return fake, nil
	}
	t.Cleanup(func() { attachmentMigrationUploaderFactory = oldFactory })

	return &migrationTestEnv{cfg: cfg, root: root, fake: fake}
}

func insertTestMessage(t *testing.T, msg *Entity.Message) {
	t.Helper()
	if err := Database.DB.Create(msg).Error; err != nil {
		t.Fatal(err)
	}
}

func channelImageMessage() *Entity.Message {
	return &Entity.Message{
		Content:   "hello",
		MessageID: 1,
		Username:  "imbGZo",
		MessageUrl: "https://t.me/imbGZo/1",
		Attachments: []Entity.Attachment{
			{
				FileName: "single.jpg",
				FilePath: "assets/imbGZo/1/single.jpg",
				FileSize: 3,
				Type:     Entity.ImageMessage,
			},
		},
	}
}

func createLocalImage(t *testing.T, env *migrationTestEnv) string {
	t.Helper()
	abs := filepath.Join(env.cfg.Output.ChannelDir, "assets", "imbGZo", "1", "single.jpg")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestBackfillAttachmentsToR2_HappyPath(t *testing.T) {
	env := setupMigrationTest(t)
	msg := channelImageMessage()
	insertTestMessage(t, msg)
	createLocalImage(t, env)

	stats, err := BackfillAttachmentsToR2(context.Background(), env.cfg)
	if err != nil {
		t.Fatalf("迁移不应失败: %v", err)
	}
	if stats.Uploaded != 1 || stats.Total != 1 || stats.MarkdownRewritten != 1 || stats.Failed != 0 || stats.Skipped != 0 {
		t.Fatalf("统计不符合预期: %+v", stats)
	}
	if len(env.fake.keys) != 1 || env.fake.keys[0] != "tg-archive/imbGZo/1/single.jpg" {
		t.Fatalf("上传 key 不匹配: %v", env.fake.keys)
	}

	var reloaded Entity.Message
	if err := Database.DB.Preload("Attachments").First(&reloaded, msg.ID).Error; err != nil {
		t.Fatal(err)
	}
	wantURL := "https://media.example.com/tg-archive/imbGZo/1/single.jpg"
	if reloaded.Attachments[0].S3Url != wantURL {
		t.Fatalf("S3Url 未回填: got %q want %q", reloaded.Attachments[0].S3Url, wantURL)
	}

	mdPath := filepath.Join(env.cfg.Output.ChannelDir, "imbGZo", "1.md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("读取重写后的 Markdown 失败: %v", err)
	}
	if !strings.Contains(string(content), "![](https://media.example.com/tg-archive/imbGZo/1/single.jpg)") {
		t.Fatalf("Markdown 未使用 R2 URL:\n%s", content)
	}
}

func TestBackfillAttachmentsToR2_SkipsNonImage(t *testing.T) {
	env := setupMigrationTest(t)
	msg := channelImageMessage()
	msg.Attachments[0].Type = Entity.TextMessage
	insertTestMessage(t, msg)

	stats, err := BackfillAttachmentsToR2(context.Background(), env.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Skipped != 1 || stats.Uploaded != 0 {
		t.Fatalf("非 image 附件应跳过: %+v", stats)
	}
	if len(env.fake.keys) != 0 {
		t.Fatalf("不应触发上传: %v", env.fake.keys)
	}
}

func TestBackfillAttachmentsToR2_SkipsAlreadyUploaded(t *testing.T) {
	env := setupMigrationTest(t)
	msg := channelImageMessage()
	msg.Attachments[0].S3Url = "https://media.example.com/already.jpg"
	insertTestMessage(t, msg)

	stats, err := BackfillAttachmentsToR2(context.Background(), env.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Skipped != 1 || stats.Uploaded != 0 {
		t.Fatalf("已上传附件应跳过: %+v", stats)
	}
	if len(env.fake.keys) != 0 {
		t.Fatalf("不应触发上传: %v", env.fake.keys)
	}
}

func TestBackfillAttachmentsToR2_HandlesUploadFailure(t *testing.T) {
	env := setupMigrationTest(t)
	env.fake.err = errors.New("boom")
	msg := channelImageMessage()
	insertTestMessage(t, msg)
	createLocalImage(t, env)

	stats, err := BackfillAttachmentsToR2(context.Background(), env.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Failed != 1 || stats.Uploaded != 0 || stats.MarkdownRewritten != 0 {
		t.Fatalf("上传失败统计不符合预期: %+v", stats)
	}

	var reloaded Entity.Message
	if err := Database.DB.Preload("Attachments").First(&reloaded, msg.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Attachments[0].S3Url != "" {
		t.Fatalf("上传失败时 S3Url 应保持空: %q", reloaded.Attachments[0].S3Url)
	}
}

func TestBackfillAttachmentsToR2_Disabled(t *testing.T) {
	env := setupMigrationTest(t)
	env.cfg.R2.Enable = false
	insertTestMessage(t, channelImageMessage())

	stats, err := BackfillAttachmentsToR2(context.Background(), env.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Uploaded != 0 || stats.Skipped != 0 {
		t.Fatalf("R2 关闭时不应做任何处理: %+v", stats)
	}
	if len(env.fake.keys) != 0 {
		t.Fatalf("R2 关闭时不应触发上传: %v", env.fake.keys)
	}
}

func TestRewriteMarkdownForMessage_ChannelPath(t *testing.T) {
	env := setupMigrationTest(t)
	msg := channelImageMessage()
	msg.Attachments[0].S3Url = "https://media.example.com/tg-archive/imbGZo/1/single.jpg"

	if err := rewriteMarkdownForMessage(env.cfg, *msg); err != nil {
		t.Fatalf("重写 Markdown 失败: %v", err)
	}

	mdPath := filepath.Join(env.cfg.Output.ChannelDir, "imbGZo", "1.md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("channel 路径未生成归档文件: %v", err)
	}
	if !strings.Contains(string(content), "![](https://media.example.com/tg-archive/imbGZo/1/single.jpg)") {
		t.Fatalf("Markdown 未包含 S3Url:\n%s", content)
	}
}

func TestRewriteMarkdownForMessage_PersonPath(t *testing.T) {
	env := setupMigrationTest(t)
	msg := &Entity.Message{
		Content:   "hello",
		MessageID: 2,
		Username:  "12345",
		Attachments: []Entity.Attachment{
			{
				FileName: "p.jpg",
				FilePath: "assets/12345/p.jpg",
				Type:     Entity.ImageMessage,
				S3Url:    "https://media.example.com/tg-archive/12345/2/p.jpg",
			},
		},
	}

	if err := rewriteMarkdownForMessage(env.cfg, *msg); err != nil {
		t.Fatalf("重写 Markdown 失败: %v", err)
	}

	mdPath := filepath.Join(env.cfg.Output.PersonDir, "12345", "2.md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("person 路径未生成归档文件: %v", err)
	}
	if !strings.Contains(string(content), "![](https://media.example.com/tg-archive/12345/2/p.jpg)") {
		t.Fatalf("Markdown 未包含 S3Url:\n%s", content)
	}
}
