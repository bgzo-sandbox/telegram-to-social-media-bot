package archiveservice

import (
	"strings"

	"telegram-message-sync-bot/internal/Entity"
)

// PreferredAttachmentURL 返回附件用于 Markdown 导出的首选 URL。
// 规则：S3Url 非空（去除首尾空白后）优先返回 R2 公开地址，否则回落本地相对路径。
// 这样做的原因是把"导出地址选择"收口在单点，实时归档、历史补录与迁移共用同一口径。
func PreferredAttachmentURL(a Entity.Attachment) string {
	if strings.TrimSpace(a.S3Url) != "" {
		return a.S3Url
	}
	return a.FilePath
}
