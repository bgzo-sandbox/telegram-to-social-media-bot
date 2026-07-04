package syncservice

import (
	"fmt"
	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/pkg/SocialMediaUtils"
)

var sendBlueSkyText = SocialMediaUtils.SendBlueSky
var sendBlueSkyImage = SocialMediaUtils.SendBlueSkyWithImage
var sendMastodonText = SocialMediaUtils.SendMastodon
var sendMastodonImage = SocialMediaUtils.SendMastodonWithImage
var sendTwitterText = SocialMediaUtils.SendTwitter
var sendTwitterImage = SocialMediaUtils.SendTwitterWithImage
var sendBlueSkyTextDetailed = SocialMediaUtils.SendBlueSkyDetailed
var sendBlueSkyImageDetailed = SocialMediaUtils.SendBlueSkyWithImageDetailed
var sendMastodonTextDetailed = SocialMediaUtils.SendMastodonDetailed
var sendMastodonImageDetailed = SocialMediaUtils.SendMastodonWithImageDetailed
var sendTwitterTextDetailed = SocialMediaUtils.SendTwitterDetailed
var sendTwitterImageDetailed = SocialMediaUtils.SendTwitterWithImageDetailed

type blueSkySender struct{}

func (blueSkySender) Name() string {
	return "BlueSky"
}

func (blueSkySender) Send(config Entity.Config, payload Payload) DispatchResult {
	prepared := PreparePlatformText("BlueSky", payload.Text)
	if payload.Image != nil && payload.Image.FilePath != "" {
		imageResult := sendBlueSkyImageDetailed(config, prepared.Text, payload.Image.FilePath)
		if imageResult.Success {
			return dispatchResultFromPublish("BlueSky", imageResult, true, true, prepared.Truncated)
		}
		textResult := sendBlueSkyTextDetailed(config, prepared.Text)
		textResult.ErrorMessage = mergeImageFallbackError(textResult, imageResult)
		return dispatchResultFromPublish("BlueSky", textResult, true, false, prepared.Truncated)
	}
	return dispatchResultFromPublish("BlueSky", sendBlueSkyTextDetailed(config, prepared.Text), false, false, prepared.Truncated)
}

type mastodonSender struct{}

func (mastodonSender) Name() string {
	return "Mastodon"
}

func (mastodonSender) Send(config Entity.Config, payload Payload) DispatchResult {
	prepared := PreparePlatformText("Mastodon", payload.Text)
	if payload.Image != nil && payload.Image.FilePath != "" {
		imageResult := sendMastodonImageDetailed(config, prepared.Text, payload.Image.FilePath)
		if imageResult.Success {
			return dispatchResultFromPublish("Mastodon", imageResult, true, true, prepared.Truncated)
		}
		textResult := sendMastodonTextDetailed(config, prepared.Text)
		textResult.ErrorMessage = mergeImageFallbackError(textResult, imageResult)
		return dispatchResultFromPublish("Mastodon", textResult, true, false, prepared.Truncated)
	}
	return dispatchResultFromPublish("Mastodon", sendMastodonTextDetailed(config, prepared.Text), false, false, prepared.Truncated)
}

type twitterSender struct{}

func (twitterSender) Name() string {
	return "Twitter"
}

func (twitterSender) Send(config Entity.Config, payload Payload) DispatchResult {
	if err := ValidateTweetText(payload.Text); err != nil {
		return DispatchResult{
			Platform:     "Twitter",
			Success:      false,
			ErrorMessage: fmt.Sprintf("twitter validation failed: %v", err),
		}
	}

	prepared := PreparePlatformText("Twitter", payload.Text)
	if payload.Image != nil && payload.Image.FilePath != "" {
		imageResult := sendTwitterImageDetailed(config, prepared.Text, payload.Image.FilePath)
		if imageResult.Success {
			return dispatchResultFromPublish("Twitter", imageResult, true, true, prepared.Truncated)
		}
		textResult := sendTwitterTextDetailed(config, prepared.Text)
		textResult.ErrorMessage = mergeImageFallbackError(textResult, imageResult)
		return dispatchResultFromPublish("Twitter", textResult, true, false, prepared.Truncated)
	}
	return dispatchResultFromPublish("Twitter", sendTwitterTextDetailed(config, prepared.Text), false, false, prepared.Truncated)
}

func DefaultSenders() []Sender {
	return []Sender{
		blueSkySender{},
		mastodonSender{},
		twitterSender{},
	}
}

func dispatchResultFromPublish(platform string, result SocialMediaUtils.PublishResult, imageRequested bool, usedImage bool, truncated bool) DispatchResult {
	return DispatchResult{
		Platform:       platform,
		Success:        result.Success,
		ImageRequested: imageRequested,
		UsedImage:      usedImage,
		Truncated:      truncated,
		RemoteID:       result.RemoteID,
		RemoteURL:      result.RemoteURL,
		ErrorMessage:   result.ErrorMessage,
	}
}

func mergeImageFallbackError(textResult SocialMediaUtils.PublishResult, imageResult SocialMediaUtils.PublishResult) string {
	if imageResult.ErrorMessage == "" {
		return textResult.ErrorMessage
	}
	if !textResult.Success {
		if textResult.ErrorMessage != "" {
			return textResult.ErrorMessage
		}
		return imageResult.ErrorMessage
	}
	return fmt.Sprintf("图片上传失败，已降级为纯文本: %s", imageResult.ErrorMessage)
}
