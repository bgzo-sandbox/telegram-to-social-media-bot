// Command fix-attachment-ext 修复历史附件文件名缺失扩展名的问题。
// 背景：buildPersistedAttachmentFileName 曾以 file_unique_id 直接作为文件名，未附加扩展名，
// 导致本地文件与 R2 object key 丢失后缀，图片无法在浏览器直接打开。
// 作用：
//  1) 按文件内容魔数识别扩展名，重命名本地文件；
//  2) 更新数据库 FileName / FilePath；
//  3) 清空已上传附件的 S3Url（旧对象无扩展名且 key 规则已变更，需重新上传）；
//  4) 修正归档 Markdown 中对旧路径/旧 S3 URL 的引用。
// 用法：go run ./scripts/migration/fix-attachment-ext -c ./config/config.yaml [-dry-run]
// 修复后请配置真实 R2 凭据并运行 ./tg migrate attachments-to-r2 重新上传。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/internal/service/bootstrapservice"
)

type fixStats struct {
	renamed    int
	s3Cleared  int
	mdUpdated  int
	unknownExt int
}

func main() {
	configFile := flag.String("c", "./config/config.yaml", "config for bot.")
	dryRun := flag.Bool("dry-run", false, "only print what would change")
	flag.Parse()

	cfg, err := bootstrapservice.LoadConfig(*configFile)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	if err := bootstrapservice.InitRuntime(cfg); err != nil {
		fmt.Printf("初始化运行时失败: %v\n", err)
		os.Exit(1)
	}

	stats, err := run(cfg, *dryRun)
	if err != nil {
		fmt.Printf("修复失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("修复完成 (dry-run=%t): %+v\n", *dryRun, stats)
}

func run(cfg Entity.Config, dryRun bool) (fixStats, error) {
	stats := fixStats{}

	var attachments []Entity.Attachment
	if err := Database.DB.Find(&attachments).Error; err != nil {
		return stats, fmt.Errorf("读取附件失败: %w", err)
	}

	bases := []string{cfg.Output.ChannelDir, cfg.Output.PersonDir}

	for _, att := range attachments {
		if att.Type != Entity.ImageMessage || hasExt(att.FileName) {
			continue
		}

		localAbs, baseIndex, err := locateFile(bases, att.FilePath)
		if err != nil {
			continue
		}

		ext, err := detectImageExt(localAbs)
		if err != nil {
			stats.unknownExt++
			fmt.Printf("跳过未知格式: id=%d file=%q (%v)\n", att.ID, att.FileName, err)
			continue
		}

		newName := att.FileName + ext
		newRel := replaceLastSegment(att.FilePath, newName)
		newAbs := filepath.Join(bases[baseIndex], newRel)

		hadS3 := strings.TrimSpace(att.S3Url) != ""

		fmt.Printf("修复: id=%d file=%q -> %q s3_cleared=%t\n", att.ID, att.FileName, newName, hadS3)

		replaced, err := fixMarkdownReferences(cfg, att.FilePath, newRel, att.S3Url, dryRun)
		if err != nil {
			return stats, fmt.Errorf("修正 Markdown 失败: %w", err)
		}
		stats.mdUpdated += replaced

		if dryRun {
			stats.renamed++
			if hadS3 {
				stats.s3Cleared++
			}
			continue
		}

		if err := os.Rename(localAbs, newAbs); err != nil {
			return stats, fmt.Errorf("重命名失败 %s -> %s: %w", localAbs, newAbs, err)
		}

		updates := map[string]interface{}{
			"file_name": newName,
			"file_path": newRel,
		}
		if hadS3 {
			updates["s3_url"] = ""
		}
		if err := Database.DB.Model(&att).Updates(updates).Error; err != nil {
			return stats, fmt.Errorf("更新附件 id=%d 失败: %w", att.ID, err)
		}

		stats.renamed++
		if hadS3 {
			stats.s3Cleared++
		}
	}

	return stats, nil
}

// locateFile 在 channel/person 目录下定位附件文件，返回绝对路径与所在 base 下标。
func locateFile(bases []string, rel string) (string, int, error) {
	for i, base := range bases {
		p := filepath.Join(base, rel)
		if _, err := os.Stat(p); err == nil {
			return p, i, nil
		}
	}
	return "", -1, fmt.Errorf("文件不存在: %s", rel)
}

// replaceLastSegment 用新文件名替换相对路径的最后一节。
func replaceLastSegment(rel string, newName string) string {
	dir := filepath.Dir(rel)
	if dir == "." {
		return newName
	}
	return filepath.Join(dir, newName)
}

// detectImageExt 按文件头魔数识别图片扩展名。
func detectImageExt(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	head := make([]byte, 16)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", err
	}
	head = head[:n]

	switch {
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return ".jpg", nil
	case len(head) >= 4 && head[0] == 0x89 && head[1] == 'P' && head[2] == 'N' && head[3] == 'G':
		return ".png", nil
	case len(head) >= 4 && head[0] == 'G' && head[1] == 'I' && head[2] == 'F' && head[3] == '8':
		return ".gif", nil
	case len(head) >= 12 && string(head[0:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return ".webp", nil
	default:
		return "", fmt.Errorf("未知图片格式")
	}
}

// fixMarkdownReferences 在归档 Markdown 中把旧相对路径与旧 S3 URL 替换为新的相对路径。
func fixMarkdownReferences(cfg Entity.Config, oldRel string, newRel string, oldS3URL string, dryRun bool) (int, error) {
	bases := []string{cfg.Output.ChannelDir, cfg.Output.PersonDir}
	replaced := 0

	for _, base := range bases {
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(content)

			changed := strings.ReplaceAll(text, oldRel, newRel)
			if oldS3URL != "" {
				changed = strings.ReplaceAll(changed, oldS3URL, newRel)
			}
			if changed == text {
				return nil
			}

			replaced++
			fmt.Printf("  markdown: %s\n", path)
			if !dryRun {
				if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return replaced, err
		}
	}

	return replaced, nil
}

// hasExt 判断文件名是否包含扩展名（最后一个 . 位于非首尾位置且中间无路径分隔符）。
func hasExt(name string) bool {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return i > 0 && i < len(name)-1
		}
		if name[i] == '/' || name[i] == '\\' {
			return false
		}
	}
	return false
}
