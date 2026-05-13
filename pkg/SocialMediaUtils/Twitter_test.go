package SocialMediaUtils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"telegram-message-sync-bot/internal/Entity"

	"github.com/michimani/gotwi"
	uploadTypes "github.com/michimani/gotwi/media/upload/types"
	"github.com/michimani/gotwi/resources"
	manageTweetTypes "github.com/michimani/gotwi/tweet/managetweet/types"
)

var testPNGBytes = []byte("\x89PNG\r\n\x1a\nrest")

func TestSendTwitterWithImage_UploadAndAttachMedia(t *testing.T) {
	originalNewClient := newTwitterClient
	originalInitialize := twitterUploadInitialize
	originalAppend := twitterUploadAppend
	originalFinalize := twitterUploadFinalize
	originalCreateTweet := twitterCreateTweet
	defer func() {
		newTwitterClient = originalNewClient
		twitterUploadInitialize = originalInitialize
		twitterUploadAppend = originalAppend
		twitterUploadFinalize = originalFinalize
		twitterCreateTweet = originalCreateTweet
	}()

	root := t.TempDir()
	imagePath := filepath.Join(root, "single.png")
	if err := os.WriteFile(imagePath, testPNGBytes, 0o644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	newTwitterClient = func(_ *gotwi.NewClientInput) (gotwi.IClient, error) {
		return gotwi.NewMockGotwiClientWithFunc(gotwi.MockFuncInput{}), nil
	}
	twitterUploadInitialize = func(client gotwi.IClient, input *uploadTypes.InitializeInput) (*uploadTypes.InitializeOutput, error) {
		if input.MediaCategory != uploadTypes.MediaCategoryTweetImage {
			t.Fatalf("unexpected media category: %+v", input)
		}
		if input.MediaType != uploadTypes.MediaTypePNG {
			t.Fatalf("unexpected media type: %+v", input)
		}
		return &uploadTypes.InitializeOutput{Data: resources.UploadedMedia{MediaID: "media-1"}}, nil
	}
	twitterUploadAppend = func(client gotwi.IClient, input *uploadTypes.AppendInput) (*uploadTypes.AppendOutput, error) {
		if input.MediaID != "media-1" {
			t.Fatalf("unexpected append input: %+v", input)
		}
		return &uploadTypes.AppendOutput{}, nil
	}
	twitterUploadFinalize = func(client gotwi.IClient, input *uploadTypes.FinalizeInput) (*uploadTypes.FinalizeOutput, error) {
		if input.MediaID != "media-1" {
			t.Fatalf("unexpected finalize input: %+v", input)
		}
		return &uploadTypes.FinalizeOutput{}, nil
	}
	twitterCreateTweet = func(client gotwi.IClient, input *manageTweetTypes.CreateInput) (*manageTweetTypes.CreateOutput, error) {
		if input.Media == nil || len(input.Media.MediaIDs) != 1 || input.Media.MediaIDs[0] != "media-1" {
			t.Fatalf("expected tweet create input to contain media id, got: %+v", input)
		}
		return &manageTweetTypes.CreateOutput{Data: struct {
			ID   *string `json:"id"`
			Text *string `json:"text"`
		}{ID: gotwi.String("1"), Text: gotwi.String("hello")}}, nil
	}

	config := Entity.Config{}
	config.SocialMediaSync.Twitter.Enable = true

	ok := SendTwitterWithImage(config, "hello", imagePath)
	if !ok {
		t.Fatalf("expected SendTwitterWithImage success")
	}
}

func TestSendTwitterWithImage_ReturnFalseWhenUploadFails(t *testing.T) {
	originalNewClient := newTwitterClient
	originalInitialize := twitterUploadInitialize
	defer func() {
		newTwitterClient = originalNewClient
		twitterUploadInitialize = originalInitialize
	}()

	root := t.TempDir()
	imagePath := filepath.Join(root, "single.png")
	if err := os.WriteFile(imagePath, testPNGBytes, 0o644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	newTwitterClient = func(_ *gotwi.NewClientInput) (gotwi.IClient, error) {
		return gotwi.NewMockGotwiClientWithFunc(gotwi.MockFuncInput{}), nil
	}
	twitterUploadInitialize = func(client gotwi.IClient, input *uploadTypes.InitializeInput) (*uploadTypes.InitializeOutput, error) {
		return nil, fmt.Errorf("init failed")
	}

	config := Entity.Config{}
	config.SocialMediaSync.Twitter.Enable = true

	ok := SendTwitterWithImage(config, "hello", imagePath)
	if ok {
		t.Fatalf("expected SendTwitterWithImage failure when upload initialize fails")
	}
}

func TestResolveTwitterMediaType(t *testing.T) {
	mediaType, err := resolveTwitterMediaType([]byte("\x89PNG\r\n\x1a\nrest"))
	if err != nil {
		t.Fatalf("expected PNG media type, got error: %v", err)
	}
	if mediaType != uploadTypes.MediaTypePNG {
		t.Fatalf("unexpected media type: %s", mediaType)
	}
}
