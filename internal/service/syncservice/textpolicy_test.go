package syncservice

import (
	"strings"
	"testing"
)

func TestPreparePlatformText_NoTruncateWithinLimit(t *testing.T) {
	prepared := PreparePlatformText("twitter", "hello")
	if prepared.Truncated {
		t.Fatalf("expected no truncation: %+v", prepared)
	}
	if prepared.Text != "hello" {
		t.Fatalf("unexpected text: %+v", prepared)
	}
}

func TestPreparePlatformText_TruncateTwitter(t *testing.T) {
	input := strings.Repeat("a", 320)
	prepared := PreparePlatformText("twitter", input)
	if !prepared.Truncated {
		t.Fatalf("expected truncation: %+v", prepared)
	}
	if twitterWeightedLength(prepared.Text) != twitterTextLimit {
		t.Fatalf("unexpected truncated size: %d", twitterWeightedLength(prepared.Text))
	}
	if !strings.HasSuffix(prepared.Text, truncateSuffix) {
		t.Fatalf("expected suffix on truncated text: %s", prepared.Text)
	}
}

func TestPreparePlatformText_NormalizeNewlinesAndEmoji(t *testing.T) {
	prepared := PreparePlatformText("twitter", " line1\r\n😀line2 ")
	if prepared.Text != "line1\n😀line2" {
		t.Fatalf("unexpected normalized text: %+v", prepared)
	}
}

func TestPreparePlatformText_TwitterWithinReasonableLimitNotTruncated(t *testing.T) {
	input := strings.Repeat("a", 240)
	prepared := PreparePlatformText("twitter", input)
	if prepared.Truncated {
		t.Fatalf("did not expect twitter truncation within 280 graphemes: %+v", prepared)
	}
	if prepared.Text != input {
		t.Fatalf("unexpected twitter text: %+v", prepared)
	}
}

func TestPreparePlatformText_MastodonWithinReasonableLimitNotTruncated(t *testing.T) {
	input := strings.Repeat("a", 450)
	prepared := PreparePlatformText("mastodon", input)
	if prepared.Truncated {
		t.Fatalf("did not expect mastodon truncation within 500 graphemes: %+v", prepared)
	}
	if prepared.Text != input {
		t.Fatalf("unexpected mastodon text: %+v", prepared)
	}
}

func TestPreparePlatformText_TwitterAllowsOneHundredFortyFullWidthCharacters(t *testing.T) {
	input := strings.Repeat("你", 140)
	prepared := PreparePlatformText("twitter", input)
	if prepared.Truncated {
		t.Fatalf("did not expect truncation at 140 full-width characters: %+v", prepared)
	}
	if twitterWeightedLength(prepared.Text) != twitterTextLimit {
		t.Fatalf("expected exact twitter limit weight, got: %d", twitterWeightedLength(prepared.Text))
	}
}

func TestPreparePlatformText_TwitterTruncatesBeyondFullWidthLimit(t *testing.T) {
	input := strings.Repeat("你", 141)
	prepared := PreparePlatformText("twitter", input)
	if !prepared.Truncated {
		t.Fatalf("expected truncation beyond 140 full-width characters: %+v", prepared)
	}
	if twitterWeightedLength(prepared.Text) > twitterTextLimit {
		t.Fatalf("unexpected twitter weighted length: %d", twitterWeightedLength(prepared.Text))
	}
	if !strings.HasSuffix(prepared.Text, truncateSuffix) {
		t.Fatalf("expected suffix on truncated text: %s", prepared.Text)
	}
}

func TestPreparePlatformText_TwitterSupportsMixedWidthBoundary(t *testing.T) {
	input := strings.Repeat("你", 100) + strings.Repeat("a", 80)
	prepared := PreparePlatformText("twitter", input)
	if prepared.Truncated {
		t.Fatalf("did not expect truncation at mixed-width twitter boundary: %+v", prepared)
	}
	if twitterWeightedLength(prepared.Text) != twitterTextLimit {
		t.Fatalf("expected exact twitter limit weight, got: %d", twitterWeightedLength(prepared.Text))
	}
}

func TestValidateTweetText_TwitterExceedsLimitWithTrailingURL(t *testing.T) {
	text := `前几年我还羡慕实体游戏的可流通性，因为我受够了 Steam 的区域定价和审查。

直到最近索尼先割席 PC，保持游戏独占，然后宣布停产光盘。车头掉转太猛了，感觉像是新官上任三把火，先把自己家烧了个遍。

如果不再有实体光盘，那么主机跟 Steam 还有什么区别呢？Why not Steam?

https://x.com/unotfbg/status/2072348056113082736`

	err := ValidateTweetText(text)
	if err == nil {
		t.Fatalf("expected validation error for text exceeding Twitter limit with trailing URL")
	}
}

func TestValidateTweetText_TwitterFitsWithURLShortening(t *testing.T) {
	text := strings.Repeat("a", 250) + " https://x.com/unotfbg/status/2072348056113082736"

	err := ValidateTweetText(text)
	if err != nil {
		t.Fatalf("expected no validation error (text fits with URL shortening): %v", err)
	}
}

func TestValidateTweetText_TwitterExceedsLimitWithoutURL(t *testing.T) {
	text := strings.Repeat("a", 300)

	err := ValidateTweetText(text)
	if err != nil {
		t.Fatalf("expected no validation error for text without URL: %v", err)
	}
}

func TestValidateTweetText_TwitterWithinLimitWithURL(t *testing.T) {
	text := "Hello world, check out this link: https://x.com/unotfbg/status/2072348056113082736"

	err := ValidateTweetText(text)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
