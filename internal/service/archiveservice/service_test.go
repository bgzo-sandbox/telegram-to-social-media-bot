package archiveservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"

	"telegram-message-sync-bot/internal/Entity"
)

func TestResolveSourceMeta_DefaultPrivateMessage(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			ID:   99,
			Chat: models.Chat{ID: 12345},
		},
	}

	config := Entity.Config{}
	config.Output.PersonDir = "/person"
	config.Output.ChannelDir = "/channel"

	meta := ResolveSourceMeta(update, config)
	if meta.OutputPath != "/person/12345" {
		t.Fatalf("expected person output path /person/12345, got: %s", meta.OutputPath)
	}
	if meta.SourceID != "12345" {
		t.Fatalf("expected sourceID=12345, got: %s", meta.SourceID)
	}
	if meta.FileName != "99.md" {
		t.Fatalf("expected fileName=99.md, got: %s", meta.FileName)
	}
	if meta.SourceLink != "" {
		t.Fatalf("expected empty source link for private message, got: %s", meta.SourceLink)
	}
}

func TestResolveSourceMeta_PrivateMessageUsesMessageDate(t *testing.T) {
	messageTime := time.Unix(1743233467, 0)
	update := &models.Update{
		Message: &models.Message{
			ID:   100,
			Date: int(messageTime.Unix()),
			Chat: models.Chat{ID: 54321},
		},
	}

	config := Entity.Config{}
	config.Output.PersonDir = "/person"
	config.Output.ChannelDir = "/channel"

	meta := ResolveSourceMeta(update, config)
	if !meta.SourceDate.Equal(messageTime) {
		t.Fatalf("expected source date %v, got %v", messageTime, meta.SourceDate)
	}
}

func TestResolveSourceMeta_ForwardedChannelWithUsername(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			ID:   1,
			Chat: models.Chat{ID: 10},
			ForwardOrigin: &models.MessageOrigin{
				Type: "channel",
				MessageOriginChannel: &models.MessageOriginChannel{
					Date:      int(time.Unix(1700000000, 0).Unix()),
					MessageID: 321,
					Chat: models.Chat{
						ID:       -1001,
						Username: "imbGZo",
					},
				},
			},
		},
	}

	config := Entity.Config{}
	config.Output.PersonDir = "/person"
	config.Output.ChannelDir = "/channel"

	meta := ResolveSourceMeta(update, config)
	if meta.OutputPath != "/channel/imbGZo" {
		t.Fatalf("expected channel output path /channel/imbGZo, got: %s", meta.OutputPath)
	}
	if meta.SourceID != "imbGZo" {
		t.Fatalf("expected sourceID=imbGZo, got: %s", meta.SourceID)
	}
	if meta.FileName != "321.md" {
		t.Fatalf("expected fileName=321.md, got: %s", meta.FileName)
	}
	if meta.SourceLink != "https://t.me/imbGZo/321" {
		t.Fatalf("unexpected source link: %s", meta.SourceLink)
	}
	if meta.MessageID != 321 {
		t.Fatalf("expected messageID=321, got: %d", meta.MessageID)
	}
}

func TestResolveArchiveFileName_DefaultTemplateFallback(t *testing.T) {
	meta := SourceMeta{MessageID: 321}
	if got := ResolveArchiveFileName(Entity.Config{}, meta, "hello world"); got != "321.md" {
		t.Fatalf("expected default file name 321.md, got: %s", got)
	}
}

func TestResolveArchiveFileName_CustomTemplate(t *testing.T) {
	config := Entity.Config{}
	config.Template.FileNameTemplate = "{{.id}}-{{.title-filename-truncated}}.md"
	meta := SourceMeta{MessageID: 321}
	content := "Hello, TG Archive! / unsafe _ title? with spaces"

	got := ResolveArchiveFileName(config, meta, content)
	if got != "321-Hello-TG-Archive.md" {
		t.Fatalf("unexpected rendered file name: %s", got)
	}
}

func TestResolveArchiveFileName_UsesMessageIDWhenSanitizedTitleEmpty(t *testing.T) {
	config := Entity.Config{}
	config.Template.FileNameTemplate = "{{.id}}-{{.title-filename-truncated}}.md"
	meta := SourceMeta{MessageID: 789}

	got := ResolveArchiveFileName(config, meta, "!!! ???")
	if got != "789-789.md" {
		t.Fatalf("expected message id fallback file name, got: %s", got)
	}
}

func TestSelectMsgText_FallbackToCaption(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			Text:    "",
			Caption: "hello #tag",
		},
	}

	text := SelectMsgText(update)
	if text != "hello \\#tag" {
		t.Fatalf("unexpected selected text: %s", text)
	}
}

func TestExtractPhotos_ReturnsForwardedUserPhotos(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			ForwardOrigin: &models.MessageOrigin{Type: "user"},
			Photo: []models.PhotoSize{
				{FileID: "small"},
				{FileID: "large"},
			},
		},
	}

	photos := extractPhotos(update)
	if len(photos) != 2 {
		t.Fatalf("expected 2 photos, got: %d", len(photos))
	}
	if photos[len(photos)-1].FileID != "large" {
		t.Fatalf("expected highest resolution photo to remain available, got: %s", photos[len(photos)-1].FileID)
	}
}

func TestBuildTemplateData_ContainsExpectedKeys(t *testing.T) {
	meta := SourceMeta{
		SourceID:   "imbGZo",
		SourceLink: "https://t.me/imbGZo/123",
		SourceDate: time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC),
		MessageID:  123,
	}
	now := time.Date(2026, 2, 17, 11, 0, 0, 0, time.UTC)

	data := BuildTemplateData(meta, "photo", "这是一个测试标题，应该截断到二十个字符之后", now)

	if data["photo"] != "photo" {
		t.Fatalf("expected photo field")
	}
	if data["content"] != "这是一个测试标题，应该截断到二十个字符之后" {
		t.Fatalf("expected content field")
	}
	if data["source"] != meta.SourceLink || data["source_message"] != meta.SourceLink || data["sourceTelegram"] != meta.SourceLink {
		t.Fatalf("expected source aliases to be populated")
	}
	if data["source_channel"] != meta.SourceID {
		t.Fatalf("expected source_channel field")
	}
	if data["published"] != "2026-02-17T10:00:00" || data["saved"] != "2026-02-17T11:00:00" || data["modified"] != "2026-02-17T11:00:00" {
		t.Fatalf("expected formatted time fields, got: %+v", data)
	}
	if data["title"] != "这是一个测试标题，应该截断到二十个字符之" {
		t.Fatalf("unexpected title: %v", data["title"])
	}
}

func TestBuildTemplateData_UsesMessageIDWhenContentEmpty(t *testing.T) {
	meta := SourceMeta{MessageID: 456}
	data := BuildTemplateData(meta, "", "", time.Date(2026, 2, 17, 11, 0, 0, 0, time.UTC))
	if data["title"] != "456" {
		t.Fatalf("expected message id fallback title, got: %v", data["title"])
	}
}

func TestNormalizeFrontMatterContent_ReplacesAllNewlines(t *testing.T) {
	input := "line1\nline2\r\nline3\rline4"
	got := normalizeFrontMatterContent(input)
	if got != "line1 line2 line3 line4" {
		t.Fatalf("unexpected normalized content: %s", got)
	}
}

func TestTruncateForFrontMatter(t *testing.T) {
	if got := truncateForFrontMatter("abc", 10); got != "abc" {
		t.Fatalf("expected original string when shorter, got: %s", got)
	}
	if got := truncateForFrontMatter("abcdef", 3); got != "abc" {
		t.Fatalf("expected truncation to 3, got: %s", got)
	}
}

func TestBuildFrontMatter_MandatoryFields(t *testing.T) {
	meta := SourceMeta{
		SourceLink: "https://t.me/channel/100",
		SourceDate: time.Date(2025, 1, 18, 16, 58, 21, 0, time.UTC),
		MessageID:  100,
	}
	archivedAt := time.Date(2025, 1, 19, 10, 20, 30, 0, time.UTC)
	content := "第一行\n第二行"

	fm := BuildFrontMatter(meta, content, archivedAt)

	checks := []string{
		"---",
		"title: \"100-第一行 第二行\"",
		"aliases:",
		"- \"100-第一行 第二行\"",
		"created: 2025-01-18T16:58:21",
		"modified: 2025-01-19T10:20:30",
		"comments: true",
		"draft: true",
		"description: \"第一行 第二行\"",
		"source: \"https://t.me/channel/100\"",
		"tags: []",
	}

	for _, check := range checks {
		if !strings.Contains(fm, check) {
			t.Fatalf("front matter missing expected snippet: %s\nactual:\n%s", check, fm)
		}
	}
}

func TestBuildFrontMatter_SourceCanBeEmpty(t *testing.T) {
	meta := SourceMeta{SourceLink: "", SourceDate: time.Date(2025, 1, 18, 16, 58, 21, 0, time.UTC), MessageID: 1}
	fm := BuildFrontMatter(meta, "hello", time.Date(2025, 1, 19, 10, 20, 30, 0, time.UTC))
	if !strings.Contains(fm, "source: \"\"") {
		t.Fatalf("expected empty source field, got:\n%s", fm)
	}
}

func TestIsMessageArchived_HitsNewPathFirst(t *testing.T) {
	root := t.TempDir()
	sourceID := "test_channel"
	messageFile := "123.md"
	sourceLink := "https://t.me/test_channel/123"

	newDir := filepath.Join(root, sourceID)
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatalf("failed to create new dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newDir, messageFile), []byte("hello\nwithout source link\n"), 0o644); err != nil {
		t.Fatalf("failed to write new archive file: %v", err)
	}

	meta := SourceMeta{
		ArchiveRoot: root,
		OutputPath:  newDir,
		FileName:    messageFile,
		SourceID:    sourceID,
		SourceLink:  sourceLink,
	}

	if !isMessageArchived(meta) {
		t.Fatalf("expected archived message to be found in new path")
	}
}

func TestIsMessageArchived_FallsBackToLegacyPath(t *testing.T) {
	root := t.TempDir()
	sourceID := "test_channel"
	messageFile := "123.md"
	sourceLink := "https://t.me/test_channel/123"

	legacyPath := filepath.Join(root, sourceID+".md")
	if err := os.WriteFile(legacyPath, []byte("legacy\n"+sourceLink+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write legacy archive file: %v", err)
	}

	meta := SourceMeta{
		ArchiveRoot: root,
		OutputPath:  filepath.Join(root, sourceID),
		FileName:    messageFile,
		SourceID:    sourceID,
		SourceLink:  sourceLink,
	}

	if !isMessageArchived(meta) {
		t.Fatalf("expected archived message to be found in legacy path fallback")
	}
}

func TestIsMessageArchived_ReturnsFalseWhenNotFound(t *testing.T) {
	root := t.TempDir()
	meta := SourceMeta{
		ArchiveRoot: root,
		OutputPath:  filepath.Join(root, "missing_source"),
		FileName:    "999.md",
		SourceID:    "missing_source",
		SourceLink:  "https://t.me/missing_source/999",
	}

	if isMessageArchived(meta) {
		t.Fatalf("expected not archived when neither new nor legacy paths contain source link")
	}
}

func TestBuildPersistedAttachmentFileName_UsesFileUniqueID(t *testing.T) {
	got := buildPersistedAttachmentFileName("AQADyxBrG0vZEVd-", "photos/image.jpg")
	if got != "AQADyxBrG0vZEVd-.jpg" {
		t.Fatalf("expected file_unique_id with remote extension, got: %s", got)
	}
}

func TestBuildPersistedAttachmentFileName_KeepsExistingExtension(t *testing.T) {
	got := buildPersistedAttachmentFileName("AQADyxBrG0vZEVd-.png", "photos/image.jpg")
	if got != "AQADyxBrG0vZEVd-.png" {
		t.Fatalf("expected existing extension to be kept, got: %s", got)
	}
}

func TestBuildPersistedAttachmentFileName_NoRemoteExtension(t *testing.T) {
	got := buildPersistedAttachmentFileName("AQADyxBrG0vZEVd-", "photos/image")
	if got != "AQADyxBrG0vZEVd-" {
		t.Fatalf("expected name unchanged when remote path has no extension, got: %s", got)
	}
}

func TestBuildPersistedAttachmentFileName_FallsBackWhenUniqueIDInvalid(t *testing.T) {
	got := buildPersistedAttachmentFileName("../", "photos/image.jpg")
	if !strings.HasSuffix(got, ".jpg") {
		t.Fatalf("expected fallback file name to keep extension, got: %s", got)
	}
	if strings.Contains(got, "/") {
		t.Fatalf("expected fallback file name to be sanitized, got: %s", got)
	}
}
