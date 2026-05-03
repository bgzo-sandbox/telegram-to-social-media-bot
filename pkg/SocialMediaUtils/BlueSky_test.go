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

func TestBuildBlueSkyPost_AddsLinkFacets(t *testing.T) {
	message := "read this https://example.com/path?x=1 and this https://bsky.app/profile/test"

	post, err := buildBlueSkyPost(message, "", "token")
	if err != nil {
		t.Fatalf("expected buildBlueSkyPost success, got: %v", err)
	}

	facets, ok := post["facets"].([]map[string]any)
	if !ok {
		t.Fatalf("expected facets slice, got: %#v", post["facets"])
	}
	if len(facets) != 2 {
		t.Fatalf("expected 2 facets, got: %d", len(facets))
	}

	firstIndex, ok := facets[0]["index"].(map[string]any)
	if !ok {
		t.Fatalf("expected first facet index, got: %#v", facets[0]["index"])
	}
	firstFeatures, ok := facets[0]["features"].([]map[string]any)
	if !ok || len(firstFeatures) != 1 {
		t.Fatalf("expected first facet features, got: %#v", facets[0]["features"])
	}
	if firstFeatures[0]["uri"] != "https://example.com/path?x=1" {
		t.Fatalf("unexpected first facet uri: %#v", firstFeatures[0]["uri"])
	}
	if firstIndex["byteStart"] != strings.Index(message, "https://example.com/path?x=1") {
		t.Fatalf("unexpected first facet start: %#v", firstIndex)
	}
}

func TestBuildBlueSkyPost_TrimsTrailingPunctuationFromLinkFacet(t *testing.T) {
	message := "see https://example.com/test."

	post, err := buildBlueSkyPost(message, "", "token")
	if err != nil {
		t.Fatalf("expected buildBlueSkyPost success, got: %v", err)
	}

	facets := post["facets"].([]map[string]any)
	features := facets[0]["features"].([]map[string]any)
	if features[0]["uri"] != "https://example.com/test" {
		t.Fatalf("unexpected facet uri: %#v", features[0]["uri"])
	}
}
