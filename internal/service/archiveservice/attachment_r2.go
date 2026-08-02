package archiveservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/pkg/S3Utils"
)

// S3Uploader 定义 R2 上传器接口，供归档层消费。
// 这样做的原因是让实时归档与测试都可以注入不同的实现。
type S3Uploader interface {
	Upload(ctx context.Context, localPath string, key string) (string, error)
}

// r2UploaderFactory 生产 S3Uploader；测试可替换为 fake。
var r2UploaderFactory = buildDefaultR2Uploader

// uploadAttachmentFactory 是 PersistMessage 调用 R2 上传的入口；测试可替换为 fake。
var uploadAttachmentFactory = uploadAttachmentToR2

// buildDefaultR2Uploader 依据配置构造默认 R2 上传器。
func buildDefaultR2Uploader(cfg Entity.Config) (S3Uploader, error) {
	return S3Utils.NewR2Uploader(cfg)
}

// uploadAttachmentToR2 上传附件到 R2，成功返回公开访问 URL。
// 这样做的原因是把"本地路径解析 + object key 构造 + 上传"收口在单点，PersistMessage 只消费结果。
func uploadAttachmentToR2(ctx context.Context, cfg Entity.Config, meta SourceMeta, file *Entity.Attachment) (string, error) {
	localPath, err := resolveLocalAttachmentPath(cfg, file.FilePath)
	if err != nil {
		return "", err
	}

	key := S3Utils.BuildR2ObjectKey(cfg.R2.Path, meta.SourceID, file.FileName)

	uploader, err := r2UploaderFactory(cfg)
	if err != nil {
		return "", err
	}

	return uploader.Upload(ctx, localPath, key)
}

// resolveLocalAttachmentPath 把附件相对路径解析为本地绝对路径。
// 顺序：绝对路径直接校验；相对路径先查 channel_dir，再查 person_dir。
func resolveLocalAttachmentPath(cfg Entity.Config, relatedPath string) (string, error) {
	if strings.TrimSpace(relatedPath) == "" {
		return "", fmt.Errorf("附件相对路径为空")
	}

	if filepath.IsAbs(relatedPath) {
		if _, err := os.Stat(relatedPath); err != nil {
			return "", err
		}
		return relatedPath, nil
	}

	candidates := []string{
		filepath.Join(cfg.Output.ChannelDir, relatedPath),
		filepath.Join(cfg.Output.PersonDir, relatedPath),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("本地附件不存在: %s", relatedPath)
}
