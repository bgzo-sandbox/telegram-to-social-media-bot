package bootstrapservice

import (
	"strings"
	"testing"

	"telegram-message-sync-bot/internal/Entity"
)

func TestValidateR2Config_Disabled_NoError(t *testing.T) {
	cfg := Entity.Config{}
	cfg.R2.Enable = false
	if err := validateR2Config(cfg); err != nil {
		t.Fatalf("R2 关闭时不应报错: %v", err)
	}
}

func TestValidateR2Config_Enabled_MissingFields(t *testing.T) {
	cfg := Entity.Config{}
	cfg.R2.Enable = true

	err := validateR2Config(cfg)
	if err == nil {
		t.Fatal("R2 开启且字段缺失时应返回错误")
	}
	msg := err.Error()
	for _, name := range []string{"server_address", "bucket", "access_key_id", "secret_access_key", "public_address"} {
		if !strings.Contains(msg, name) {
			t.Fatalf("缺失字段 %s 未在错误信息中列出: %s", name, msg)
		}
	}
}

func TestValidateR2Config_Enabled_AllFields(t *testing.T) {
	cfg := Entity.Config{}
	cfg.R2.Enable = true
	cfg.R2.ServerAddress = "https://account.r2.cloudflarestorage.com"
	cfg.R2.Bucket = "bucket"
	cfg.R2.AccessKeyID = "kid"
	cfg.R2.SecretAccessKey = "sak"
	cfg.R2.PublicAddress = "https://media.example.com"

	if err := validateR2Config(cfg); err != nil {
		t.Fatalf("字段齐全时不应报错: %v", err)
	}
}
