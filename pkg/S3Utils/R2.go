package S3Utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"telegram-message-sync-bot/internal/Entity"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Uploader 定义 R2 对象上传能力的稳定接口，供服务层消费并便于测试注入 fake。
type Uploader interface {
	Upload(ctx context.Context, localPath string, key string) (publicURL string, err error)
}

// R2Uploader 是 Cloudflare R2（S3 兼容）上传器的具体实现。
// 使用静态凭证 + 路径风格寻址（R2 仅支持 path-style），Region 固定为 "auto"。
type R2Uploader struct {
	client        *s3.Client
	bucket        string
	publicAddress string
}

// NewR2Uploader 根据配置构造 R2 上传器。
// 这样做的原因是把 AWS SDK 的装配细节收口在单点，禁止调用方自行拼装 aws.Config。
func NewR2Uploader(cfg Entity.Config) (*R2Uploader, error) {
	if strings.TrimSpace(cfg.R2.ServerAddress) == "" {
		return nil, fmt.Errorf("r2 server_address 不能为空")
	}
	if strings.TrimSpace(cfg.R2.Bucket) == "" {
		return nil, fmt.Errorf("r2 bucket 不能为空")
	}

	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.R2.AccessKeyID, cfg.R2.SecretAccessKey, ""))

	client := s3.New(s3.Options{
		Region:       "auto",
		Credentials:  creds,
		BaseEndpoint: aws.String(strings.TrimRight(cfg.R2.ServerAddress, "/")),
		UsePathStyle: true,
	})

	return &R2Uploader{
		client:        client,
		bucket:        cfg.R2.Bucket,
		publicAddress: cfg.R2.PublicAddress,
	}, nil
}

// Upload 读取本地文件并上传到 R2，成功返回公开访问 URL。
// 错误信息经过脱敏处理，只保留 service error code 与 message，不回显请求体或密钥。
func (u *R2Uploader) Upload(ctx context.Context, localPath string, key string) (string, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return "", err
	}

	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return "", sanitizeS3Error(err)
	}

	return BuildPublicURL(u.publicAddress, key), nil
}

// BuildPublicURL 拼接公开访问 URL：<public_address>/<key>，并对首尾斜杠做归一化。
func BuildPublicURL(publicAddress string, key string) string {
	base := strings.TrimRight(publicAddress, "/")
	trimmedKey := strings.Trim(key, "/")
	if trimmedKey == "" {
		return base
	}
	return base + "/" + trimmedKey
}

// sanitizeS3Error 把 AWS SDK 错误脱敏为可安全落日志的形式。
func sanitizeS3Error(err error) error {
	var apiErr interface {
		ErrorCode() string
		ErrorMessage() string
	}
	if errors.As(err, &apiErr) {
		return fmt.Errorf("r2 service error: code=%s message=%s", apiErr.ErrorCode(), apiErr.ErrorMessage())
	}
	return fmt.Errorf("r2 upload error: %w", err)
}
