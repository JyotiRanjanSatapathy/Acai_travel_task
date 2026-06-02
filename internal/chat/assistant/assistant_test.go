package assistant

import (
	"context"
	"testing"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
)

func TestAssistant_Title(t *testing.T) {
	a := &Assistant{}

	t.Run("empty conversation", func(t *testing.T) {
		title, err := a.Title(context.Background(), &model.Conversation{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if title != "An empty conversation" {
			t.Errorf("Title() = %q, want %q", title, "An empty conversation")
		}
	})
}
