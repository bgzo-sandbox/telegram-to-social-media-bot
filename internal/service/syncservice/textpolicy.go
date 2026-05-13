package syncservice

import (
	"strings"

	"github.com/rivo/uniseg"
)

const (
	twitterTextLimit  = 100
	mastodonTextLimit = 300
	blueSkyTextLimit  = 300
	truncateSuffix    = "..."
)

type PreparedText struct {
	Text      string
	Truncated bool
}

func PreparePlatformText(platform string, text string) PreparedText {
	normalized := normalizePlatformText(text)
	limit := platformTextLimit(platform)
	if limit <= 0 {
		return PreparedText{Text: normalized}
	}

	graphemes := splitGraphemes(normalized)
	if len(graphemes) <= limit {
		return PreparedText{Text: normalized}
	}

	suffixGraphemes := splitGraphemes(truncateSuffix)
	bodyLimit := limit - len(suffixGraphemes)
	if bodyLimit < 0 {
		bodyLimit = 0
	}

	return PreparedText{
		Text:      strings.Join(append(graphemes[:bodyLimit], suffixGraphemes...), ""),
		Truncated: true,
	}
}

func normalizePlatformText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimSpace(text)
}

func platformTextLimit(platform string) int {
	switch NormalizePlatform(platform) {
	case "twitter":
		return twitterTextLimit
	case "mastodon":
		return mastodonTextLimit
	case "bluesky":
		return blueSkyTextLimit
	default:
		return 0
	}
}

func splitGraphemes(text string) []string {
	iter := uniseg.NewGraphemes(text)
	parts := make([]string, 0, len(text))
	for iter.Next() {
		parts = append(parts, iter.Str())
	}
	return parts
}
