package tui

import (
	"strings"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func TestFormatSendResult_message(t *testing.T) {
	msg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("hello"))
	text := formatSendResult(msg)
	if text != "hello" {
		t.Fatalf("text = %q", text)
	}
}

func TestFormatChatError_methodNotFoundHint(t *testing.T) {
	wrapped := a2a.NewError(a2a.ErrMethodNotFound, "Method not found")
	out := formatChatError(wrapped)
	if !strings.Contains(out, "SendMessage") || !strings.Contains(out, "supportedInterfaces") {
		t.Fatalf("expected hint in %q", out)
	}
}
