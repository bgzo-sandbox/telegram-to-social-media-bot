package syncservice

import (
	"strings"

	"github.com/rivo/uniseg"
	"golang.org/x/text/width"
)

const (
	twitterTextLimit  = 280
	mastodonTextLimit = 500
	blueSkyTextLimit  = 300
	truncateSuffix    = "..."
)

type PreparedText struct {
	Text      string
	Truncated bool
}

func PreparePlatformText(platform string, text string) PreparedText {
	normalized := normalizePlatformText(text)
	if NormalizePlatform(platform) == "twitter" {
		return prepareTwitterText(normalized)
	}

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

func prepareTwitterText(text string) PreparedText {
	if twitterWeightedLength(text) <= twitterTextLimit {
		return PreparedText{Text: text}
	}

	bodyLimit := twitterTextLimit - twitterWeightedLength(truncateSuffix)
	if bodyLimit < 0 {
		bodyLimit = 0
	}

	graphemes := splitGraphemes(text)
	var builder strings.Builder
	used := 0
	for _, grapheme := range graphemes {
		weight := twitterGraphemeWeight(grapheme)
		if used+weight > bodyLimit {
			break
		}
		builder.WriteString(grapheme)
		used += weight
	}
	builder.WriteString(truncateSuffix)

	return PreparedText{
		Text:      builder.String(),
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

func twitterWeightedLength(text string) int {
	total := 0
	for _, grapheme := range splitGraphemes(text) {
		total += twitterGraphemeWeight(grapheme)
	}
	return total
}

func twitterGraphemeWeight(grapheme string) int {
	if grapheme == "" {
		return 0
	}

	for _, r := range grapheme {
		switch width.LookupRune(r).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			return 2
		}
	}

	return 1
}
