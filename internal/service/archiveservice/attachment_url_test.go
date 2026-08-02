package archiveservice

import (
	"testing"

	"telegram-message-sync-bot/internal/Entity"
)

func TestPreferredAttachmentURL_PreferS3(t *testing.T) {
	a := Entity.Attachment{
		FilePath: "assets/imbGZo/single.jpg",
		S3Url:    "https://media.example.com/imbGZo/1/single.jpg",
	}
	if got := PreferredAttachmentURL(a); got != a.S3Url {
		t.Fatalf("存在 S3Url 时应优先返回 S3Url, got %q", got)
	}
}

func TestPreferredAttachmentURL_FallbackToLocal(t *testing.T) {
	a := Entity.Attachment{
		FilePath: "assets/imbGZo/single.jpg",
		S3Url:    "",
	}
	if got := PreferredAttachmentURL(a); got != a.FilePath {
		t.Fatalf("S3Url 为空时应回落本地路径, got %q", got)
	}
}

func TestPreferredAttachmentURL_EmptyS3SpacesTreatedAsMissing(t *testing.T) {
	a := Entity.Attachment{
		FilePath: "assets/imbGZo/single.jpg",
		S3Url:    "   ",
	}
	if got := PreferredAttachmentURL(a); got != a.FilePath {
		t.Fatalf("S3Url 全空白应视为缺失并回落本地路径, got %q", got)
	}
}
