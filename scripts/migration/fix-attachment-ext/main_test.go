package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectImageExt_JPG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "img")
	if err := os.WriteFile(path, []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := detectImageExt(path)
	if err != nil || got != ".jpg" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestDetectImageExt_PNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "img")
	if err := os.WriteFile(path, []byte{0x89, 'P', 'N', 'G', 0x0D}, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := detectImageExt(path)
	if err != nil || got != ".png" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestDetectImageExt_GIF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "img")
	if err := os.WriteFile(path, []byte("GIF89a"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := detectImageExt(path)
	if err != nil || got != ".gif" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestDetectImageExt_WEBP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "img")
	if err := os.WriteFile(path, append([]byte("RIFF"), append([]byte{0, 0, 0, 0}, []byte("WEBP")...)...), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := detectImageExt(path)
	if err != nil || got != ".webp" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestDetectImageExt_Unknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "img")
	if err := os.WriteFile(path, []byte("plain text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := detectImageExt(path); err == nil {
		t.Fatal("未知格式应返回错误")
	}
}

func TestHasExt(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"AQADFQxrG9_RmFR8", false},
		{"AQADFQxrG9_RmFR8.jpg", true},
		{".hidden", false},
		{"a.tar.gz", true},
		{"dir/a.b/c", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := hasExt(tc.name); got != tc.want {
			t.Fatalf("hasExt(%q) = %t, want %t", tc.name, got, tc.want)
		}
	}
}

func TestReplaceLastSegment(t *testing.T) {
	if got := replaceLastSegment("assets/SomeACG/AQADFQxrG9_RmFR8", "AQADFQxrG9_RmFR8.jpg"); got != "assets/SomeACG/AQADFQxrG9_RmFR8.jpg" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := replaceLastSegment("AQADFQxrG9_RmFR8", "AQADFQxrG9_RmFR8.jpg"); got != "AQADFQxrG9_RmFR8.jpg" {
		t.Fatalf("unexpected: %q", got)
	}
}
