package attachmentmigrationservice

import (
	"context"
	"fmt"
	"strings"

	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/internal/service/syncservice"
	"telegram-message-sync-bot/pkg/LogUtils"
	"telegram-message-sync-bot/pkg/S3Utils"
)

// BackfillStats 记录历史附件迁移的统计结果。
type BackfillStats struct {
	Total             int
	Skipped           int
	Uploaded          int
	Failed            int
	MarkdownRewritten int
}

// attachmentMigrationUploaderFactory 生产 R2 上传器；测试可替换为 fake。
var attachmentMigrationUploaderFactory = func(cfg Entity.Config) (S3Utils.Uploader, error) {
	return S3Utils.NewR2Uploader(cfg)
}

// BackfillAttachmentsToR2 把历史图片附件批量上传到 R2，回填 S3Url 并重写对应 Markdown。
// 规则：仅处理 Type==Image 且 S3Url 为空的附件；幂等可重复执行。
// 这样做的原因是让历史数据与实时归档最终收敛到同一"优先 R2 URL"口径。
func BackfillAttachmentsToR2(ctx context.Context, cfg Entity.Config) (BackfillStats, error) {
	stats := BackfillStats{}
	if !cfg.R2.Enable {
		return stats, nil
	}

	msgs, err := Database.ListMessages()
	if err != nil {
		return stats, fmt.Errorf("读取数据库消息失败: %w", err)
	}

	uploader, err := attachmentMigrationUploaderFactory(cfg)
	if err != nil {
		return stats, fmt.Errorf("初始化 R2 上传器失败: %w", err)
	}

	for i := range msgs {
		msg := &msgs[i]
		for j := range msg.Attachments {
			attachment := &msg.Attachments[j]
			if attachment.Type != Entity.ImageMessage || strings.TrimSpace(attachment.S3Url) != "" {
				stats.Skipped++
				continue
			}

			localPath := syncservice.ResolvePayloadImagePath(cfg, attachment.FilePath)
			if localPath == "" {
				stats.Failed++
				LogUtils.GetLogger().Printf("附件迁移: 本地文件缺失, file=%s\n", attachment.FilePath)
				continue
			}

			key := S3Utils.BuildR2ObjectKey(cfg.R2.Path, msg.Username, msg.MessageID, attachment.FileName)
			url, err := uploader.Upload(ctx, localPath, key)
			if err != nil {
				stats.Failed++
				LogUtils.GetLogger().Printf("附件迁移: 上传失败, key=%s err=%v\n", key, err)
				continue
			}

			attachment.S3Url = url
			stats.Total++
			stats.Uploaded++

			if err := Database.DB.Model(attachment).Update("S3Url", url).Error; err != nil {
				return stats, fmt.Errorf("回填 S3Url 失败: %w", err)
			}

			if err := rewriteMarkdownForMessage(cfg, *msg); err != nil {
				return stats, fmt.Errorf("重写 Markdown 失败: %w", err)
			}
			stats.MarkdownRewritten++
		}
	}

	return stats, nil
}
