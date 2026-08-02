package S3Utils

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"telegram-message-sync-bot/internal/Entity"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func testConfig() Entity.Config {
	cfg := Entity.Config{}
	cfg.R2.Enable = true
	cfg.R2.ServerAddress = "https://account.r2.cloudflarestorage.com"
	cfg.R2.Bucket = "test-bucket"
	cfg.R2.AccessKeyID = "AKID123456"
	cfg.R2.SecretAccessKey = "SECRET456789"
	cfg.R2.Path = ""
	cfg.R2.PublicAddress = "https://media.example.com"
	return cfg
}

func TestNewR2Uploader_ValidConfig(t *testing.T) {
	cfg := testConfig()
	uploader, err := NewR2Uploader(cfg)
	if err != nil {
		t.Fatalf("NewR2Uploader 不应返回错误: %v", err)
	}
	if uploader.bucket != cfg.R2.Bucket {
		t.Fatalf("bucket 不匹配: got %q want %q", uploader.bucket, cfg.R2.Bucket)
	}
	if uploader.publicAddress != cfg.R2.PublicAddress {
		t.Fatalf("publicAddress 不匹配: got %q want %q", uploader.publicAddress, cfg.R2.PublicAddress)
	}
}

func TestNewR2Uploader_MissingServerAddress(t *testing.T) {
	cfg := testConfig()
	cfg.R2.ServerAddress = " "
	if _, err := NewR2Uploader(cfg); err == nil {
		t.Fatal("server_address 为空时应返回错误")
	}
}

func TestNewR2Uploader_MissingBucket(t *testing.T) {
	cfg := testConfig()
	cfg.R2.Bucket = ""
	if _, err := NewR2Uploader(cfg); err == nil {
		t.Fatal("bucket 为空时应返回错误")
	}
}

func TestR2Uploader_Upload_Success(t *testing.T) {
	var capturedBody []byte
	var capturedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("期望 PUT 请求, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "test-bucket") {
			t.Fatalf("请求路径应包含 bucket: %s", r.URL.Path)
		}
		capturedBody, _ = io.ReadAll(r.Body)
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	uploader := &R2Uploader{
		client: s3.New(s3.Options{
			Region:       "auto",
			Credentials:  credentials.NewStaticCredentialsProvider("AKID123456", "SECRET456789", ""),
			BaseEndpoint: aws.String(server.URL),
			UsePathStyle: true,
		}),
		bucket:        "test-bucket",
		publicAddress: "https://media.example.com",
	}

	localPath := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(localPath, []byte("image-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	url, err := uploader.Upload(context.Background(), localPath, "assets/imbGZo/42/photo.jpg")
	if err != nil {
		t.Fatalf("Upload 不应返回错误: %v", err)
	}
	if url != "https://media.example.com/assets/imbGZo/42/photo.jpg" {
		t.Fatalf("公开 URL 不匹配: got %q", url)
	}
	if string(capturedBody) != "image-bytes" {
		t.Fatalf("上传体不匹配: got %q", string(capturedBody))
	}
	if capturedContentType != "image/jpeg" {
		t.Fatalf("Content-Type 应按扩展名推断为 image/jpeg, got %q", capturedContentType)
	}
}

func TestR2Uploader_Upload_UnknownExt_FallsBackToSniffing(t *testing.T) {
	var capturedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	uploader := &R2Uploader{
		client: s3.New(s3.Options{
			Region:       "auto",
			Credentials:  credentials.NewStaticCredentialsProvider("AKID123456", "SECRET456789", ""),
			BaseEndpoint: aws.String(server.URL),
			UsePathStyle: true,
		}),
		bucket:        "test-bucket",
		publicAddress: "https://media.example.com",
	}

	localPath := filepath.Join(t.TempDir(), "photo") // 无扩展名
	// 0xFF 0xD8 0xFF 为 JPEG 魔数头
	if err := os.WriteFile(localPath, []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := uploader.Upload(context.Background(), localPath, "assets/k/photo"); err != nil {
		t.Fatalf("Upload 不应返回错误: %v", err)
	}
	if capturedContentType != "image/jpeg" {
		t.Fatalf("无扩展名时应按文件头探测为 image/jpeg, got %q", capturedContentType)
	}
}

func TestR2Uploader_Upload_MissingLocal(t *testing.T) {
	uploader := &R2Uploader{
		client: s3.New(s3.Options{
			Region:       "auto",
			Credentials:  credentials.NewStaticCredentialsProvider("AKID123456", "SECRET456789", ""),
			BaseEndpoint: aws.String("http://127.0.0.1:9"),
			UsePathStyle: true,
		}),
		bucket:        "test-bucket",
		publicAddress: "https://media.example.com",
	}

	_, err := uploader.Upload(context.Background(), filepath.Join(t.TempDir(), "not-exist.jpg"), "assets/k")
	if err == nil {
		t.Fatal("本地文件不存在时应返回错误")
	}
}

func TestR2Uploader_BuildPublicURL_TrimSlashes(t *testing.T) {
	cases := []struct {
		name          string
		publicAddress string
		key           string
		want          string
	}{
		{"both clean", "https://media.example.com", "assets/a.jpg", "https://media.example.com/assets/a.jpg"},
		{"public trailing slash", "https://media.example.com/", "assets/a.jpg", "https://media.example.com/assets/a.jpg"},
		{"key leading slash", "https://media.example.com", "/assets/a.jpg", "https://media.example.com/assets/a.jpg"},
		{"both slashes", "https://media.example.com//", "/assets/a.jpg", "https://media.example.com/assets/a.jpg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BuildPublicURL(tc.publicAddress, tc.key); got != tc.want {
				t.Fatalf("BuildPublicURL(%q, %q) = %q, want %q", tc.publicAddress, tc.key, got, tc.want)
			}
		})
	}
}

func TestR2Uploader_Upload_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	uploader := &R2Uploader{
		client: s3.New(s3.Options{
			Region:       "auto",
			Credentials:  credentials.NewStaticCredentialsProvider("AKID123456", "SECRET456789", ""),
			BaseEndpoint: aws.String(server.URL),
			UsePathStyle: true,
		}),
		bucket:        "test-bucket",
		publicAddress: "https://media.example.com",
	}

	localPath := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(localPath, []byte("image-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := uploader.Upload(context.Background(), localPath, "assets/k/photo.jpg")
	if err == nil {
		t.Fatal("服务端 503 时应返回错误")
	}
	msg := err.Error()
	if strings.Contains(msg, "AKID123456") || strings.Contains(msg, "SECRET456789") {
		t.Fatalf("错误信息不得包含密钥明文: %s", msg)
	}
}
