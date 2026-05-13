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
	input := strings.Repeat("a", 120)
	prepared := PreparePlatformText("twitter", input)
	if !prepared.Truncated {
		t.Fatalf("expected truncation: %+v", prepared)
	}
	if len(splitGraphemes(prepared.Text)) != twitterTextLimit {
		t.Fatalf("unexpected truncated size: %d", len(splitGraphemes(prepared.Text)))
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
