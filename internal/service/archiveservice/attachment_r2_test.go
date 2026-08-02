package archiveservice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"telegram-message-sync-bot/internal/Entity"
)

type fakeS3Uploader struct {
	key string
	url string
	err error
}

func (f *fakeS3Uploader) Upload(_ context.Context, localPath string, key string) (string, error) {
	f.key = key
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

func TestUploadAttachmentToR2_LocalPathNotFound(t *testing.T) {
	cfg := Entity.Config{}
	cfg.Output.ChannelDir = t.TempDir()
	cfg.Output.PersonDir = t.TempDir()
	cfg.R2.Path = "tg-archive"

	meta := SourceMeta{SourceID: "imbGZo", MessageID: 42}
	file := &Entity.Attachment{FileName: "photo.jpg", FilePath: "assets/imbGZo/photo.jpg", Type: Entity.ImageMessage}

	oldFactory := r2UploaderFactory
	defer func() { r2UploaderFactory = oldFactory }()
	r2UploaderFactory = func(cfg Entity.Config) (S3Uploader, error) {
		t.Fatal("本地路径不存在时不应构造 uploader")
		return nil, nil
	}

	url, err := uploadAttachmentToR2(context.Background(), cfg, meta, file)
	if err == nil {
		t.Fatal("本地附件不存在时应返回错误")
	}
	if url != "" {
		t.Fatalf("失败时不应返回 URL, got %q", url)
	}
}

func TestUploadAttachmentToR2_UploaderFailure(t *testing.T) {
	channelDir := t.TempDir()
	localAbs := filepath.Join(channelDir, "assets", "imbGZo", "photo.jpg")
	if err := os.MkdirAll(filepath.Dir(localAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localAbs, []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Entity.Config{}
	cfg.Output.ChannelDir = channelDir
	cfg.Output.PersonDir = t.TempDir()
	cfg.R2.Path = "tg-archive"

	meta := SourceMeta{SourceID: "imbGZo", MessageID: 42}
	file := &Entity.Attachment{FileName: "photo.jpg", FilePath: "assets/imbGZo/photo.jpg", Type: Entity.ImageMessage}

	oldFactory := r2UploaderFactory
	defer func() { r2UploaderFactory = oldFactory }()
	r2UploaderFactory = func(cfg Entity.Config) (S3Uploader, error) {
		return &fakeS3Uploader{err: errors.New("boom")}, nil
	}

	url, err := uploadAttachmentToR2(context.Background(), cfg, meta, file)
	if err == nil {
		t.Fatal("上传失败时应返回错误")
	}
	if url != "" {
		t.Fatalf("上传失败时不应返回 URL, got %q", url)
	}
}

func TestUploadAttachmentToR2_Success(t *testing.T) {
	channelDir := t.TempDir()
	localAbs := filepath.Join(channelDir, "assets", "imbGZo", "photo.jpg")
	if err := os.MkdirAll(filepath.Dir(localAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localAbs, []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Entity.Config{}
	cfg.Output.ChannelDir = channelDir
	cfg.Output.PersonDir = t.TempDir()
	cfg.R2.Path = "tg-archive"

	meta := SourceMeta{SourceID: "imbGZo", MessageID: 42}
	file := &Entity.Attachment{FileName: "photo.jpg", FilePath: "assets/imbGZo/photo.jpg", Type: Entity.ImageMessage}

	fake := &fakeS3Uploader{url: "https://media.example.com/tg-archive/imbGZo/42/photo.jpg"}
	oldFactory := r2UploaderFactory
	defer func() { r2UploaderFactory = oldFactory }()
	r2UploaderFactory = func(cfg Entity.Config) (S3Uploader, error) {
		return fake, nil
	}

	url, err := uploadAttachmentToR2(context.Background(), cfg, meta, file)
	if err != nil {
		t.Fatalf("上传不应返回错误: %v", err)
	}
	if url != fake.url {
		t.Fatalf("返回 URL 与 fake 不一致: got %q want %q", url, fake.url)
	}
	if fake.key != "tg-archive/imbGZo/42/photo.jpg" {
		t.Fatalf("object key 不匹配: got %q", fake.key)
	}
}

func TestResolveLocalAttachmentPath_AbsPathFirst(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "img.jpg")
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Entity.Config{}
	cfg.Output.ChannelDir = t.TempDir()
	cfg.Output.PersonDir = t.TempDir()

	got, err := resolveLocalAttachmentPath(cfg, abs)
	if err != nil {
		t.Fatalf("绝对路径应解析成功: %v", err)
	}
	if got != abs {
		t.Fatalf("解析结果不一致: got %q want %q", got, abs)
	}
}

func TestResolveLocalAttachmentPath_EmptyPath(t *testing.T) {
	cfg := Entity.Config{}
	if _, err := resolveLocalAttachmentPath(cfg, "  "); err == nil {
		t.Fatal("空相对路径应返回错误")
	}
}
