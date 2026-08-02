package S3Utils

import (
	"path/filepath"
	"strings"
)

// BuildR2ObjectKey 构造 R2 对象键：<path>/<source_id>/<filename>。
// 这样做的原因是把 key 命名规则收口在单点，保证实时上传与历史迁移生成一致的键。
// 每段先 TrimSpace 再 Trim "/"，空段跳过，最终键不以 "/" 开头或结尾。
func BuildR2ObjectKey(pathPrefix string, sourceID string, filename string) string {
	segments := []string{
		strings.Trim(strings.TrimSpace(pathPrefix), "/"),
		strings.Trim(strings.TrimSpace(sourceID), "/"),
		filepath.Base(filename),
	}

	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		trimmed := strings.Trim(segment, "/")
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}

	return strings.Join(parts, "/")
}
