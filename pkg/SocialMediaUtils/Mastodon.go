package SocialMediaUtils

import (
	"context"
	"fmt"
	"log"
	"telegram-message-sync-bot/internal/Entity"

	"github.com/mattn/go-mastodon"
)

type mastodonClient interface {
	UploadMedia(ctx context.Context, file string) (*mastodon.Attachment, error)
	PostStatus(ctx context.Context, toot *mastodon.Toot) (*mastodon.Status, error)
}

var newMastodonClient = func(config *mastodon.Config) mastodonClient {
	return mastodon.NewClient(config)
}

func initMastodon(config Entity.Config) mastodon.Config {
	Mastodon := config.SocialMediaSync.Mastodon
	return mastodon.Config{
		Server:       Mastodon.Instance,
		ClientID:     Mastodon.ClientId,
		ClientSecret: Mastodon.ClientSecret,
		AccessToken:  Mastodon.AccessToken,
	}
}

func SendMastodon(globalConfig Entity.Config, Message string) bool {
	return SendMastodonDetailed(globalConfig, Message).Success
}

func SendMastodonDetailed(globalConfig Entity.Config, Message string) PublishResult {
	if globalConfig.SocialMediaSync.Mastodon.Enable == false {
		log.Println("Mastodon is not enabled in the configuration.")
		return PublishResult{ErrorMessage: "Mastodon is not enabled in the configuration."}
	}

	config := initMastodon(globalConfig)
	return postMastodon(newMastodonClient(&config), Message, "")
}

func SendMastodonWithImage(globalConfig Entity.Config, Message string, imagePath string) bool {
	return SendMastodonWithImageDetailed(globalConfig, Message, imagePath).Success
}

func SendMastodonWithImageDetailed(globalConfig Entity.Config, Message string, imagePath string) PublishResult {
	if globalConfig.SocialMediaSync.Mastodon.Enable == false {
		log.Println("Mastodon is not enabled in the configuration.")
		return PublishResult{ErrorMessage: "Mastodon is not enabled in the configuration."}
	}

	config := initMastodon(globalConfig)
	return postMastodon(newMastodonClient(&config), Message, imagePath)
}

func postMastodon(client mastodonClient, message string, imagePath string) PublishResult {
	if client == nil {
		return PublishResult{ErrorMessage: "mastodon client is nil"}
	}

	visibility := "public"

	toot := mastodon.Toot{
		Status:     message,
		Visibility: visibility,
	}

	if imagePath != "" {
		attachment, err := client.UploadMedia(context.Background(), imagePath)
		if err != nil {
			log.Println(err)
			return PublishResult{ErrorMessage: err.Error()}
		}
		toot.MediaIDs = []mastodon.ID{attachment.ID}
	}

	post, err := client.PostStatus(context.Background(), &toot)
	if err != nil {
		log.Println(err)
		return PublishResult{ErrorMessage: err.Error()}
	}

	fmt.Println("My new post is:", post)
	return PublishResult{Success: true, RemoteID: string(post.ID), RemoteURL: post.URL}
}
