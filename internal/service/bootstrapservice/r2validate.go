package bootstrapservice

import (
	"fmt"
	"strings"

	"telegram-message-sync-bot/internal/Entity"
)

// validateR2Config 校验 R2 配置必填项。
// 规则：r2.enable 关闭时直接通过；开启时 server_address / bucket / access_key_id / secret_access_key / public_address 均不可为空。
// 这样做的原因是把配置校验提前到启动阶段，避免运行期上传时才暴露缺字段。
func validateR2Config(cfg Entity.Config) error {
	if !cfg.R2.Enable {
		return nil
	}

	required := []struct {
		name  string
		value string
	}{
		{"server_address", cfg.R2.ServerAddress},
		{"bucket", cfg.R2.Bucket},
		{"access_key_id", cfg.R2.AccessKeyID},
		{"secret_access_key", cfg.R2.SecretAccessKey},
		{"public_address", cfg.R2.PublicAddress},
	}

	var missing []string
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("r2 配置缺失: %s", strings.Join(missing, ","))
	}

	return nil
}
