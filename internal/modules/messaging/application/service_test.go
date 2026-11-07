package application

import (
	"strings"
	"testing"

	"mars-admin-go/internal/modules/messaging/domain"
)

func TestNormalizeMessageRejectsEmptyAndOversized(t *testing.T) {
	input := domain.SendInput{Content: "   "}
	if err := normalizeMessage(&input); err == nil {
		t.Fatal("expected empty content to be rejected")
	}
	input.Content = strings.Repeat("a", 10001)
	if err := normalizeMessage(&input); err == nil {
		t.Fatal("expected oversized content to be rejected")
	}
}

func TestNormalizeMessageDefaultsType(t *testing.T) {
	input := domain.SendInput{Content: "hello"}
	if err := normalizeMessage(&input); err != nil {
		t.Fatal(err)
	}
	if input.MessageType != "text" {
		t.Fatalf("unexpected type %q", input.MessageType)
	}
}
