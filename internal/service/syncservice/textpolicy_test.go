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
