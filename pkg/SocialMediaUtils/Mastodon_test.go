package SocialMediaUtils

import (
	"context"
	"fmt"
	"testing"

	"telegram-message-sync-bot/internal/Entity"

	"github.com/mattn/go-mastodon"
)

type fakeMastodonClient struct {
	uploadedPath string
	postedToot   *mastodon.Toot
	uploadErr    error
	postErr      error
}

func (f *fakeMastodonClient) UploadMedia(_ context.Context, file string) (*mastodon.Attachment, error) {
	f.uploadedPath = file
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	return &mastodon.Attachment{ID: "attachment-1"}, nil
}

func (f *fakeMastodonClient) PostStatus(_ context.Context, toot *mastodon.Toot) (*mastodon.Status, error) {
	f.postedToot = toot
	if f.postErr != nil {
		return nil, f.postErr
	}
	return &mastodon.Status{}, nil
}

func TestSendMastodonWithImage_UploadAndAttachMedia(t *testing.T) {
	client := &fakeMastodonClient{}
	originalFactory := newMastodonClient
	newMastodonClient = func(_ *mastodon.Config) mastodonClient {
		return client
	}
	defer func() {
		newMastodonClient = originalFactory
	}()

	config := Entity.Config{}
	config.SocialMediaSync.Mastodon.Enable = true

	ok := SendMastodonWithImage(config, "hello", "/tmp/test.jpg")
	if !ok {
		t.Fatalf("expected SendMastodonWithImage success")
	}
	if client.uploadedPath != "/tmp/test.jpg" {
		t.Fatalf("unexpected uploaded path: %s", client.uploadedPath)
	}
	if client.postedToot == nil {
		t.Fatalf("expected toot to be posted")
	}
	if client.postedToot.Status != "hello" {
		t.Fatalf("unexpected toot status: %+v", client.postedToot)
	}
	if len(client.postedToot.MediaIDs) != 1 || client.postedToot.MediaIDs[0] != mastodon.ID("attachment-1") {
		t.Fatalf("unexpected media ids: %+v", client.postedToot.MediaIDs)
	}
}

func TestSendMastodonWithImage_ReturnFalseWhenUploadFails(t *testing.T) {
	client := &fakeMastodonClient{uploadErr: fmt.Errorf("upload failed")}
	originalFactory := newMastodonClient
	newMastodonClient = func(_ *mastodon.Config) mastodonClient {
		return client
	}
	defer func() {
		newMastodonClient = originalFactory
	}()

	config := Entity.Config{}
	config.SocialMediaSync.Mastodon.Enable = true

	ok := SendMastodonWithImage(config, "hello", "/tmp/test.jpg")
	if ok {
		t.Fatalf("expected SendMastodonWithImage failure when upload fails")
	}
	if client.postedToot != nil {
		t.Fatalf("expected post not to be attempted when upload fails")
	}
}
