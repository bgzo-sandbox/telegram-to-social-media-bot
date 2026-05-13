package syncservice

import (
	"os"
	"path/filepath"
	"telegram-message-sync-bot/internal/Entity"
	"testing"
)

type fakeSender struct {
	name    string
	success bool
	payload Payload
}

func (f fakeSender) Name() string {
	return f.name
}

func (f *fakeSender) Send(_ Entity.Config, payload Payload) DispatchResult {
	f.payload = payload
	return DispatchResult{Platform: f.name, Success: f.success, ImageRequested: payload.Image != nil, UsedImage: payload.Image != nil && f.success}
}

func TestShouldSync_Disabled(t *testing.T) {
	config := Entity.Config{}
	config.SocialMediaSync.Enable = false

	ok, reason := ShouldSync(config, "imbGZo")
	if ok {
		t.Fatalf("expected false when sync disabled")
	}
	if reason == "" {
		t.Fatalf("expected non-empty reason when sync disabled")
	}
}

func TestShouldSync_EmptyTargetChannels(t *testing.T) {
	config := Entity.Config{}
	config.SocialMediaSync.Enable = true

	ok, reason := ShouldSync(config, "imbGZo")
	if ok {
		t.Fatalf("expected false when targetChannel is empty")
	}
	if reason == "" {
		t.Fatalf("expected non-empty reason when targetChannel is empty")
	}
}

func TestShouldSync_HitTargetChannel(t *testing.T) {
	config := Entity.Config{}
	config.SocialMediaSync.Enable = true
	config.SocialMediaSync.TargetChannel = []string{"imbGZo", "another"}

	ok, reason := ShouldSync(config, "imbGZo")
	if !ok {
		t.Fatalf("expected true when source matches targetChannel, got reason: %s", reason)
	}
	if reason != "" {
		t.Fatalf("expected empty reason on match, got: %s", reason)
	}
}

func TestShouldSync_MissTargetChannel(t *testing.T) {
	config := Entity.Config{}
	config.SocialMediaSync.Enable = true
	config.SocialMediaSync.TargetChannel = []string{"imbGZo"}

	ok, reason := ShouldSync(config, "other")
	if ok {
		t.Fatalf("expected false when source does not match targetChannel")
	}
	if reason == "" {
		t.Fatalf("expected non-empty reason when source does not match")
	}
}

func TestDispatch_KeepOrderAndResults(t *testing.T) {
	config := Entity.Config{}
	senders := []Sender{
		&fakeSender{name: "BlueSky", success: true},
		&fakeSender{name: "Mastodon", success: false},
		nil,
		&fakeSender{name: "Twitter", success: true},
	}

	results := Dispatch(config, Payload{Text: "hello", Image: &ImagePayload{FilePath: "/tmp/test.jpg"}}, senders)
	if len(results) != 3 {
		t.Fatalf("expected 3 results (nil sender skipped), got %d", len(results))
	}

	if results[0].Platform != "BlueSky" || !results[0].Success {
		t.Fatalf("unexpected first result: %+v", results[0])
	}

	if results[1].Platform != "Mastodon" || results[1].Success {
		t.Fatalf("unexpected second result: %+v", results[1])
	}

	if results[2].Platform != "Twitter" || !results[2].Success {
		t.Fatalf("unexpected third result: %+v", results[2])
	}
	if !results[0].ImageRequested || !results[0].UsedImage {
		t.Fatalf("expected first result to record image usage: %+v", results[0])
	}
}

func TestDispatch_PassPayloadToSender(t *testing.T) {
	config := Entity.Config{}
	sender := &fakeSender{name: "BlueSky", success: true}
	payload := Payload{Text: "hello", Image: &ImagePayload{FilePath: "/tmp/test.jpg"}}

	results := Dispatch(config, payload, []Sender{sender})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("unexpected results: %+v", results)
	}
	if sender.payload.Text != "hello" {
		t.Fatalf("unexpected payload text: %+v", sender.payload)
	}
	if sender.payload.Image == nil || sender.payload.Image.FilePath != "/tmp/test.jpg" {
		t.Fatalf("unexpected payload image: %+v", sender.payload)
	}
}

func TestBuildPayload_NoImagePath(t *testing.T) {
	payload := BuildPayload("hello", "")
	if payload.Text != "hello" {
		t.Fatalf("unexpected payload text: %+v", payload)
	}
	if payload.Image != nil {
		t.Fatalf("expected nil image for empty path, got: %+v", payload)
	}
}

func TestBuildPayload_MissingImagePathFallbackToTextOnly(t *testing.T) {
	payload := BuildPayload("hello", "/tmp/not-found.jpg")
	if payload.Text != "hello" {
		t.Fatalf("unexpected payload text: %+v", payload)
	}
	if payload.Image != nil {
		t.Fatalf("expected nil image when file does not exist, got: %+v", payload)
	}
}

func TestBuildPayload_KeepSingleExistingImage(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "single.jpg")
	if err := os.WriteFile(imagePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	payload := BuildPayload("", imagePath)
	if payload.Text != "" {
		t.Fatalf("expected empty text payload, got: %+v", payload)
	}
	if payload.Image == nil || payload.Image.FilePath != imagePath {
		t.Fatalf("unexpected payload image: %+v", payload)
	}
}
