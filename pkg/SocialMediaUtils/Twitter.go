package SocialMediaUtils

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"telegram-message-sync-bot/internal/Entity"

	"github.com/michimani/gotwi/media/upload"
	uploadTypes "github.com/michimani/gotwi/media/upload/types"
	"github.com/michimani/gotwi/tweet/managetweet"
	manageTweetTypes "github.com/michimani/gotwi/tweet/managetweet/types"

	"github.com/michimani/gotwi"
)

var newTwitterClient = func(config *gotwi.NewClientInput) (gotwi.IClient, error) {
	return gotwi.NewClient(config)
}

var twitterCreateTweet = func(client gotwi.IClient, input *manageTweetTypes.CreateInput) (*manageTweetTypes.CreateOutput, error) {
	return managetweet.Create(context.Background(), client, input)
}

var twitterUploadInitialize = func(client gotwi.IClient, input *uploadTypes.InitializeInput) (*uploadTypes.InitializeOutput, error) {
	return upload.Initialize(context.Background(), client, input)
}

var twitterUploadAppend = func(client gotwi.IClient, input *uploadTypes.AppendInput) (*uploadTypes.AppendOutput, error) {
	return upload.Append(context.Background(), client, input)
}

var twitterUploadFinalize = func(client gotwi.IClient, input *uploadTypes.FinalizeInput) (*uploadTypes.FinalizeOutput, error) {
	return upload.Finalize(context.Background(), client, input)
}

func initTwitter(config Entity.Config) gotwi.NewClientInput {

	Twitter := config.SocialMediaSync.Twitter
	return gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           Twitter.OauthToken,
		OAuthTokenSecret:     Twitter.OauthTokenSecret,
	}
}

func SendTwitter(globalConfig Entity.Config, Message string) bool {
	return SendTwitterDetailed(globalConfig, Message).Success
}

func SendTwitterWithImage(globalConfig Entity.Config, Message string, imagePath string) bool {
	return SendTwitterWithImageDetailed(globalConfig, Message, imagePath).Success
}

func SendTwitterDetailed(globalConfig Entity.Config, message string) PublishResult {
	return sendTwitterPostDetailed(globalConfig, message, "")
}

func SendTwitterWithImageDetailed(globalConfig Entity.Config, message string, imagePath string) PublishResult {
	return sendTwitterPostDetailed(globalConfig, message, imagePath)
}

func sendTwitterPostDetailed(globalConfig Entity.Config, message string, imagePath string) PublishResult {
	// 提前返回结果失败
	if globalConfig.SocialMediaSync.Twitter.Enable == false {
		log.Println("Twitter is not enabled in the configuration.")
		return PublishResult{ErrorMessage: "Twitter is not enabled in the configuration."}
	}

	config := initTwitter(globalConfig)
	/**
	* more config for
	* GOTWI_API_KEY
	* GOTWI_API_KEY_SECRET
	 */

	c, err := newTwitterClient(&config)
	if err != nil {
		fmt.Println(err)
		return PublishResult{ErrorMessage: err.Error()}
	}

	p := &manageTweetTypes.CreateInput{
		Text: gotwi.String(message),
	}

	if imagePath != "" {
		mediaID, err := uploadTwitterImage(c, imagePath)
		if err != nil {
			fmt.Println(err)
			return PublishResult{ErrorMessage: err.Error()}
		}
		p.Media = &manageTweetTypes.CreateInputMedia{
			MediaIDs: []string{mediaID},
		}
	}

	res, err := twitterCreateTweet(c, p)
	if err != nil {
		fmt.Println(err.Error())
		return PublishResult{ErrorMessage: err.Error()}
	}

	remoteID := gotwi.StringValue(res.Data.ID)
	fmt.Printf("[%s] %s\n", remoteID, gotwi.StringValue(res.Data.Text))
	return PublishResult{
		Success:   true,
		RemoteID:  remoteID,
		RemoteURL: fmt.Sprintf("https://twitter.com/i/web/status/%s", remoteID),
	}
}

func uploadTwitterImage(client gotwi.IClient, imagePath string) (string, error) {
	fileBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	mediaType, err := resolveTwitterMediaType(fileBytes)
	if err != nil {
		return "", err
	}

	initRes, err := twitterUploadInitialize(client, &uploadTypes.InitializeInput{
		MediaType:     mediaType,
		TotalBytes:    len(fileBytes),
		Shared:        false,
		MediaCategory: uploadTypes.MediaCategoryTweetImage,
	})
	if err != nil {
		return "", err
	}

	mediaID := initRes.Data.MediaID
	if _, err := twitterUploadAppend(client, &uploadTypes.AppendInput{
		MediaID:      mediaID,
		Media:        bytes.NewReader(fileBytes),
		SegmentIndex: 0,
	}); err != nil {
		return "", err
	}

	if _, err := twitterUploadFinalize(client, &uploadTypes.FinalizeInput{MediaID: mediaID}); err != nil {
		return "", err
	}

	return mediaID, nil
}

func resolveTwitterMediaType(fileBytes []byte) (uploadTypes.MediaType, error) {
	switch http.DetectContentType(fileBytes) {
	case string(uploadTypes.MediaTypeJPEG):
		return uploadTypes.MediaTypeJPEG, nil
	case string(uploadTypes.MediaTypePNG):
		return uploadTypes.MediaTypePNG, nil
	case string(uploadTypes.MediaTypeGIF):
		return uploadTypes.MediaTypeGIF, nil
	case string(uploadTypes.MediaTypeWebP):
		return uploadTypes.MediaTypeWebP, nil
	default:
		return "", os.ErrInvalid
	}
}
