package chat

import (
	"context"
	"testing"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	. "github.com/acai-travel/tech-challenge/internal/chat/testing"
	"github.com/acai-travel/tech-challenge/internal/pb"
	"github.com/google/go-cmp/cmp"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/testing/protocmp"
)

// mockAssistant is a deterministic Assistant used in tests to avoid calling
// OpenAI. Title and Reply return the configured values, or the configured
// errors when set.
type mockAssistant struct {
	title    string
	reply    string
	titleErr error
	replyErr error
}

func (m *mockAssistant) Title(_ context.Context, _ *model.Conversation) (string, error) {
	return m.title, m.titleErr
}

func (m *mockAssistant) Reply(_ context.Context, _ *model.Conversation) (string, error) {
	return m.reply, m.replyErr
}

func TestServer_DescribeConversation(t *testing.T) {
	ctx := context.Background()
	srv := NewServer(model.New(ConnectMongo()), nil)

	t.Run("describe existing conversation", WithFixture(func(t *testing.T, f *Fixture) {
		c := f.CreateConversation()

		out, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: c.ID.Hex()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, want := out.GetConversation(), c.Proto()
		if !cmp.Equal(got, want, protocmp.Transform()) {
			t.Errorf("DescribeConversation() mismatch (-got +want):\n%s", cmp.Diff(got, want, protocmp.Transform()))
		}
	}))

	t.Run("describe non existing conversation should return 404", WithFixture(func(t *testing.T, f *Fixture) {
		_, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: "08a59244257c872c5943e2a2"})
		if err == nil {
			t.Fatal("expected error for non-existing conversation, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.NotFound {
			t.Fatalf("expected twirp.NotFound error, got %v", err)
		}
	}))
}

func TestServer_StartConversation(t *testing.T) {
	ctx := context.Background()

	t.Run("creates conversation with title and reply", WithFixture(func(t *testing.T, f *Fixture) {
		assist := &mockAssistant{title: "Weather in Barcelona", reply: "It is sunny today."}
		srv := NewServer(f.Repository, assist)

		out, err := srv.StartConversation(ctx, &pb.StartConversationRequest{Message: "What is the weather like in Barcelona?"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if out.GetConversationId() == "" {
			t.Fatal("expected a non-empty conversation ID")
		}
		t.Cleanup(func() {
			if err := f.Repository.DeleteConversation(ctx, out.GetConversationId()); err != nil {
				t.Logf("failed to cleanup conversation %s: %v", out.GetConversationId(), err)
			}
		})

		if out.GetTitle() != assist.title {
			t.Errorf("title = %q, want %q", out.GetTitle(), assist.title)
		}

		if out.GetReply() != assist.reply {
			t.Errorf("reply = %q, want %q", out.GetReply(), assist.reply)
		}

		// Verify the conversation was persisted with the user message and the
		// assistant reply.
		conv, err := f.Repository.DescribeConversation(ctx, out.GetConversationId())
		if err != nil {
			t.Fatalf("failed to describe persisted conversation: %v", err)
		}

		if conv.Title != assist.title {
			t.Errorf("persisted title = %q, want %q", conv.Title, assist.title)
		}

		if len(conv.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(conv.Messages))
		}

		if conv.Messages[0].Role != model.RoleUser || conv.Messages[0].Content != "What is the weather like in Barcelona?" {
			t.Errorf("unexpected user message: %+v", conv.Messages[0])
		}

		if conv.Messages[1].Role != model.RoleAssistant || conv.Messages[1].Content != assist.reply {
			t.Errorf("unexpected assistant message: %+v", conv.Messages[1])
		}
	}))

	t.Run("empty message returns required argument error", WithFixture(func(t *testing.T, f *Fixture) {
		srv := NewServer(f.Repository, &mockAssistant{})

		_, err := srv.StartConversation(ctx, &pb.StartConversationRequest{Message: "   "})
		if err == nil {
			t.Fatal("expected error for empty message, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.InvalidArgument {
			t.Fatalf("expected twirp.InvalidArgument error, got %v", err)
		}
	}))
}
