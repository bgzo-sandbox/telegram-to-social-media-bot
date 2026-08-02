package attachmentmigrationservice

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/internal/service/archiveservice"
)

// resolveArchiveRoot 按消息是否有来源链接决定归档根目录：channel_dir 优先，否则 person_dir。
// 规则与 archivemigrationservice.resolveArchiveRoot 保持一致。
func resolveArchiveRoot(cfg Entity.Config, msg Entity.Message) string {
	if msg.MessageUrl != "" {
		return cfg.Output.ChannelDir
	}
	return cfg.Output.PersonDir
}

// resolveArchiveSourceDirName 在归档根目录下定位与 sourceID 大小写不敏感的目录名。
// 规则与 archivemigrationservice.resolveArchiveSourceDirName 保持一致。
func resolveArchiveSourceDirName(archiveRoot string, sourceID string) (string, error) {
	entries, err := os.ReadDir(archiveRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return sourceID, nil
		}
		return "", fmt.Errorf("读取归档目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.EqualFold(entry.Name(), sourceID) {
			return entry.Name(), nil
		}
	}

	return sourceID, nil
}

// renderAttachmentMarkdown 把附件列表渲染为 Markdown 图片片段，优先使用 S3Url。
// 规则与 archivemigrationservice.renderAttachmentMarkdown 保持一致。
func renderAttachmentMarkdown(assets []Entity.Attachment) string {
	if len(assets) == 0 {
		return ""
	}

	var b strings.Builder
	for _, a := range assets {
		preferred := archiveservice.PreferredAttachmentURL(a)
		if preferred == "" {
			continue
		}
		b.WriteString("![](")
		b.WriteString(preferred)
		b.WriteString(") ")
	}
	return b.String()
}

// rewriteMarkdownForMessage 依据数据库消息重新渲染并覆盖写回归档 Markdown 文件。
// 这样做的原因是迁移 S3Url 后立即让历史 Markdown 使用新的公开 URL，避免留待二次手工处理。
func rewriteMarkdownForMessage(cfg Entity.Config, msg Entity.Message) error {
	tmplData, err := os.ReadFile(cfg.Template.Dir)
	if err != nil {
		return fmt.Errorf("读取模板失败: %w", err)
	}
	tmpl, err := template.New("archive").Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	meta := archiveservice.SourceMeta{
		SourceLink: msg.MessageUrl,
		SourceDate: msg.MessageDate,
		SourceID:   msg.Username,
		MessageID:  int(msg.MessageID),
	}

	fileName := archiveservice.ResolveArchiveFileName(cfg, meta, msg.Content)

	archiveRoot := resolveArchiveRoot(cfg, msg)
	sourceDirName, err := resolveArchiveSourceDirName(archiveRoot, meta.SourceID)
	if err != nil {
		return err
	}
	outputDir := filepath.Join(archiveRoot, sourceDirName)

	tplData := archiveservice.BuildTemplateData(
		meta,
		renderAttachmentMarkdown(msg.Attachments),
		msg.Content,
		msg.CreatedTime,
	)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tplData); err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}
	content := strings.TrimLeft(buf.String(), "\n")

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	fullPath := filepath.Join(outputDir, fileName)
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入归档文件失败: %w", err)
	}

	return nil
}
