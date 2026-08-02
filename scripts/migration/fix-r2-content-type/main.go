// Command fix-r2-content-type 一次性修复 R2 存量对象的 Content-Type。
// 背景：R2Uploader.Upload 早期未设置 Content-Type，R2 默认将对象存为
// application/octet-stream，导致浏览器直接下载图片而非内联预览。
// 作用：遍历桶内对象，对 Content-Type 为 application/octet-stream（或缺失）的对象，
// 按对象键扩展名推断 MIME 类型，并通过 CopyObject（MetadataDirective=REPLACE）覆盖更新。
// 用法：go run ./scripts/migration/fix-r2-content-type -c ./config/config.yaml [-dry-run]
package main

import (
	"context"
	"flag"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/internal/service/bootstrapservice"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type fixStats struct {
	scanned int
	fixed   int
	skipped int
	failed  int
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

	client, err := newS3Client(cfg)
	if err != nil {
		fmt.Printf("初始化 R2 客户端失败: %v\n", err)
		os.Exit(1)
	}

	stats, err := run(context.Background(), client, cfg.R2.Bucket, *dryRun)
	if err != nil {
		fmt.Printf("修复失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("修复完成 (dry-run=%t): scanned=%d fixed=%d skipped=%d failed=%d\n",
		*dryRun, stats.scanned, stats.fixed, stats.skipped, stats.failed)
}

func newS3Client(cfg Entity.Config) (*s3.Client, error) {
	if strings.TrimSpace(cfg.R2.ServerAddress) == "" {
		return nil, fmt.Errorf("r2 server_address 不能为空")
	}

	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.R2.AccessKeyID, cfg.R2.SecretAccessKey, ""))

	return s3.New(s3.Options{
		Region:       "auto",
		Credentials:  creds,
		BaseEndpoint: aws.String(strings.TrimRight(cfg.R2.ServerAddress, "/")),
		UsePathStyle: true,
	}), nil
}

func run(ctx context.Context, client *s3.Client, bucket string, dryRun bool) (fixStats, error) {
	stats := fixStats{}
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return stats, fmt.Errorf("列出对象失败: %w", err)
		}
		for _, obj := range page.Contents {
			stats.scanned++
			if err := fixObject(ctx, client, bucket, obj, dryRun, &stats); err != nil {
				stats.failed++
				fmt.Printf("失败: key=%s (%v)\n", aws.ToString(obj.Key), err)
			}
		}
	}

	return stats, nil
}

func fixObject(ctx context.Context, client *s3.Client, bucket string, obj types.Object, dryRun bool, stats *fixStats) error {
	key := aws.ToString(obj.Key)

	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	current := aws.ToString(head.ContentType)
	if current != "" && current != "application/octet-stream" {
		stats.skipped++
		return nil
	}

	target := inferContentType(key)
	if target == "" || target == current {
		stats.skipped++
		return nil
	}

	fmt.Printf("修复: key=%s content_type=%q -> %q\n", key, current, target)
	if dryRun {
		stats.fixed++
		return nil
	}

	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:            aws.String(bucket),
		CopySource:        aws.String(bucket + "/" + escapeKey(key)),
		Key:               aws.String(key),
		ContentType:       aws.String(target),
		MetadataDirective: types.MetadataDirectiveReplace,
	})
	if err != nil {
		return err
	}

	stats.fixed++
	return nil
}

// inferContentType 按对象键扩展名推断 MIME 类型；无扩展名时返回空串表示不处理。
func inferContentType(key string) string {
	return mime.TypeByExtension(filepath.Ext(key))
}

// escapeKey 对对象键的每个路径段做 URL 编码，用于 CopyObject 的 CopySource。
// 这样做的原因是避免键含空格等特殊字符时被 R2 解析错误。
func escapeKey(key string) string {
	segments := strings.Split(key, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
