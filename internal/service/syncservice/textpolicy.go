package syncservice

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rivo/uniseg"
	"golang.org/x/text/width"
)

const (
	twitterTextLimit  = 280
	twitterURLLength  = 25
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

type urlSegment struct {
	text  string
	isURL bool
}

func splitURLSegments(text string) []urlSegment {
	locs := urlRegex.FindAllStringIndex(text, -1)
	if len(locs) == 0 {
		return []urlSegment{{text: text}}
	}

	segs := make([]urlSegment, 0, len(locs)*2+1)
	pos := 0
	for _, loc := range locs {
		if loc[0] > pos {
			segs = append(segs, urlSegment{text: text[pos:loc[0]]})
		}
		segs = append(segs, urlSegment{text: text[loc[0]:loc[1]], isURL: true})
		pos = loc[1]
	}
	if pos < len(text) {
		segs = append(segs, urlSegment{text: text[pos:]})
	}
	return segs
}

func prepareTwitterText(text string) PreparedText {
	if twitterEffectiveLength(text) <= twitterTextLimit {
		return PreparedText{Text: text}
	}

	bodyLimit := twitterTextLimit - twitterWeightedLength(truncateSuffix)
	if bodyLimit < 0 {
		bodyLimit = 0
	}

	segs := splitURLSegments(text)

	totalWeight := 0
	for _, s := range segs {
		if s.isURL {
			totalWeight += twitterURLLength
		} else {
			totalWeight += twitterWeightedLength(s.text)
		}
	}
	if totalWeight <= bodyLimit {
		return PreparedText{Text: text}
	}

	excess := totalWeight - bodyLimit
	addedSuffix := false

	for i := len(segs) - 1; i >= 0 && excess > 0; i-- {
		if segs[i].isURL {
			continue
		}

		graphemes := splitGraphemes(segs[i].text)
		if len(graphemes) == 0 {
			continue
		}

		weights := make([]int, len(graphemes))
		segWeight := 0
		for j, g := range graphemes {
			w := twitterGraphemeWeight(g)
			weights[j] = w
			segWeight += w
		}

		remove := excess
		if remove > segWeight {
			remove = segWeight
		}

		removed := 0
		keepIdx := len(graphemes)
		for j := len(graphemes) - 1; j >= 0; j-- {
			if removed+weights[j] >= remove {
				removed += weights[j]
				keepIdx = j
				break
			}
			removed += weights[j]
			keepIdx = j
		}

		if keepIdx <= 0 {
			segs[i].text = ""
		} else {
			segs[i].text = strings.Join(graphemes[:keepIdx], "")
			if !addedSuffix {
				segs[i].text += truncateSuffix
				addedSuffix = true
			}
		}

		excess -= removed
	}

	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.text)
	}

	return PreparedText{Text: b.String(), Truncated: true}
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

var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

func containsURL(text string) bool {
	return urlRegex.MatchString(text)
}

func twitterEffectiveLength(text string) int {
	result := urlRegex.ReplaceAllStringFunc(text, func(match string) string {
		return strings.Repeat("x", twitterURLLength)
	})
	return twitterWeightedLength(result)
}

func ValidateTweetText(text string) error {
	normalized := normalizePlatformText(text)

	effectiveLen := twitterEffectiveLength(normalized)
	if effectiveLen <= twitterTextLimit {
		return nil
	}

	bodyLimit := twitterTextLimit - twitterWeightedLength(truncateSuffix)
	graphemes := splitGraphemes(normalized)

	used := 0
	truncateGraphemeIdx := len(graphemes)
	for i, g := range graphemes {
		w := twitterGraphemeWeight(g)
		if used+w > bodyLimit {
			truncateGraphemeIdx = i
			break
		}
		used += w
	}

	if truncateGraphemeIdx >= len(graphemes) {
		return nil
	}

	truncateBytePos := 0
	for i := 0; i < truncateGraphemeIdx; i++ {
		truncateBytePos += len(graphemes[i])
	}

	locs := urlRegex.FindAllStringIndex(normalized, -1)
	for _, loc := range locs {
		urlStart, urlEnd := loc[0], loc[1]
		if urlStart >= truncateBytePos || (urlStart < truncateBytePos && urlEnd > truncateBytePos) {
			return fmt.Errorf(
				"text exceeds Twitter limit (%d effective chars) and truncation would break a URL",
				effectiveLen,
			)
		}
	}

	return nil
}
