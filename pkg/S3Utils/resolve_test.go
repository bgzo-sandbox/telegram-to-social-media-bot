package S3Utils

import "testing"

func TestBuildR2ObjectKey_AllSegments(t *testing.T) {
	got := BuildR2ObjectKey("tg-archive", "imbGZo", "photo.jpg")
	want := "tg-archive/imbGZo/photo.jpg"
	if got != want {
		t.Fatalf("BuildR2ObjectKey 全段组合 = %q, want %q", got, want)
	}
}

func TestBuildR2ObjectKey_NoPathPrefix(t *testing.T) {
	got := BuildR2ObjectKey("", "imbGZo", "photo.jpg")
	want := "imbGZo/photo.jpg"
	if got != want {
		t.Fatalf("BuildR2ObjectKey 无 path 前缀 = %q, want %q", got, want)
	}
}

func TestBuildR2ObjectKey_EmptySourceID(t *testing.T) {
	got := BuildR2ObjectKey("tg-archive", "  ", "photo.jpg")
	want := "tg-archive/photo.jpg"
	if got != want {
		t.Fatalf("BuildR2ObjectKey 空 source_id = %q, want %q", got, want)
	}
}

func TestBuildR2ObjectKey_LeadingTrailingSlashes(t *testing.T) {
	got := BuildR2ObjectKey("/tg-archive/", "/imbGZo/", "photo.jpg")
	want := "tg-archive/imbGZo/photo.jpg"
	if got != want {
		t.Fatalf("BuildR2ObjectKey 首尾斜杠 = %q, want %q", got, want)
	}
}

func TestBuildR2ObjectKey_FileNameWithPath(t *testing.T) {
	got := BuildR2ObjectKey("tg-archive", "imbGZo", "dir/sub/photo.jpg")
	want := "tg-archive/imbGZo/photo.jpg"
	if got != want {
		t.Fatalf("BuildR2ObjectKey 带路径文件名 = %q, want %q", got, want)
	}
}

func TestBuildR2ObjectKey_WhitespaceSegments(t *testing.T) {
	got := BuildR2ObjectKey("  ", "  imbGZo  ", "photo.jpg")
	want := "imbGZo/photo.jpg"
	if got != want {
		t.Fatalf("BuildR2ObjectKey 空白段 = %q, want %q", got, want)
	}
}
