package SocialMediaUtils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestBuildBlueSkyPost_WithImageEmbed(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "single.png")
	if err := os.WriteFile(imagePath, []byte("png-data"), 0o644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	originalClient := blueSkyHTTPClient
	blueSkyHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != blueSkyUploadBlobURL {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		if req.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("unexpected auth header: %s", req.Header.Get("Authorization"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"blob":{"$type":"blob","mimeType":"image/png","size":8}}`)),
			Header:     make(http.Header),
		}, nil
	})}
	defer func() {
		blueSkyHTTPClient = originalClient
	}()

	post, err := buildBlueSkyPost("hello", imagePath, "token")
	if err != nil {
		t.Fatalf("expected buildBlueSkyPost success, got: %v", err)
	}
	if post["text"] != "hello" {
		t.Fatalf("unexpected post text: %+v", post)
	}
	embed, ok := post["embed"].(map[string]any)
	if !ok {
		t.Fatalf("expected embed map, got: %#v", post["embed"])
	}
	if embed["$type"] != "app.bsky.embed.images" {
		t.Fatalf("unexpected embed type: %+v", embed)
	}
}

func TestBuildBlueSkyPost_ReturnErrorWhenUploadFails(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "single.png")
	if err := os.WriteFile(imagePath, []byte("png-data"), 0o644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	originalClient := blueSkyHTTPClient
	blueSkyHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("upload failed")
	})}
	defer func() {
		blueSkyHTTPClient = originalClient
	}()

	_, err := buildBlueSkyPost("hello", imagePath, "token")
	if err == nil {
		t.Fatalf("expected buildBlueSkyPost failure when upload fails")
	}
}
