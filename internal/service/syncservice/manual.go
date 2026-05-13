package syncservice

import (
	"fmt"
	"strings"
	"sync"

	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
)

type ManualResyncResult struct {
	Requested bool
	Reason    string
	Results   []DispatchResult
}

var manualDispatchFactory = DefaultSenders
var manualDispatchGuard = newInFlightGuard()

type inFlightGuard struct {
	mu       sync.Mutex
	inFlight map[string]struct{}
}

func newInFlightGuard() *inFlightGuard {
	return &inFlightGuard{inFlight: make(map[string]struct{})}
}

func (g *inFlightGuard) Acquire(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.inFlight[key]; exists {
		return false
	}
	g.inFlight[key] = struct{}{}
	return true
}

func (g *inFlightGuard) Release(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.inFlight, key)
}

func ManualResync(config Entity.Config, archivedMessageID int64, platform string) (ManualResyncResult, error) {
	msg, err := Database.GetMessageByID(archivedMessageID)
	if err != nil {
		return ManualResyncResult{}, err
	}

	if !ContainsExactTarget(config.SocialMediaSync.TargetChannel, msg.Username) {
		return ManualResyncResult{Requested: false, Reason: "该消息来源不在可同步频道内"}, nil
	}

	senders, err := resolveManualSenders(platform)
	if err != nil {
		return ManualResyncResult{}, err
	}

	guardKey := fmt.Sprintf("%d:%s", archivedMessageID, NormalizePlatform(platform))
	if !manualDispatchGuard.Acquire(guardKey) {
		return ManualResyncResult{Requested: false, Reason: "相同范围的重同步正在执行，请稍后再试"}, nil
	}
	defer manualDispatchGuard.Release(guardKey)

	imagePath := firstImagePath(msg.Attachments)
	payload := BuildPayload(msg.Content, imagePath)
	results := Dispatch(config, payload, senders)
	if err := PersistDispatchResults(archivedMessageID, results, DispatchTriggerManual); err != nil {
		return ManualResyncResult{}, err
	}

	return ManualResyncResult{Requested: true, Results: results}, nil
}

func resolveManualSenders(platform string) ([]Sender, error) {
	if NormalizePlatform(platform) == "all" {
		return manualDispatchFactory(), nil
	}

	selected := make([]Sender, 0, 1)
	for _, sender := range manualDispatchFactory() {
		if NormalizePlatform(sender.Name()) == NormalizePlatform(platform) {
			selected = append(selected, sender)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("未知平台: %s", platform)
	}
	return selected, nil
}

func NormalizePlatform(platform string) string {
	platform = strings.TrimSpace(strings.ToLower(platform))
	switch platform {
	case "bs", "bluesky":
		return "bluesky"
	case "md", "mastodon":
		return "mastodon"
	case "tw", "twitter":
		return "twitter"
	case "all":
		return "all"
	default:
		return platform
	}
}

func firstImagePath(attachments []Entity.Attachment) string {
	for _, attachment := range attachments {
		if attachment.Type == Entity.ImageMessage {
			return attachment.FilePath
		}
	}
	return ""
}
