package assistant

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli   openai.Client
	tools []Tool
}

func New() *Assistant {
	cli := openai.NewClient()
	tools := []Tool{
		weatherTool{},
		forecastTool{},
		dateTool{},
		holidaysTool{},
		itineraryTool{cli: cli},
	}
	return &Assistant{cli: cli, tools: tools}
}

func (a *Assistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "An empty conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation", "conversation_id", conv.ID)

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(
			"You are a title generator. Reply with ONLY a short topic title (3-6 words) " +
				"for the user's message. Never answer, explain, or address the message. " +
				"No surrounding quotes, no trailing punctuation, no emojis or special characters.",
		),
		openai.UserMessage("Generate a title for this conversation message:\n\n" + conv.Messages[0].Content),
	}

	resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4_1Mini,
		Messages: msgs,
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	title := resp.Choices[0].Message.Content
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Trim(title, " \t\r\n-\"'")

	if len(title) > 80 {
		title = title[:80]
	}

	return title, nil
}

func (a *Assistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation", "conversation_id", conv.ID)

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(
			"You are a helpful, concise AI assistant. Provide accurate, safe, and clear responses. " +
				"When a user asks for weather, forecast, current date or time, holidays, or a day-by-day travel plan, " +
				"use the relevant tool instead of inventing details. After a tool returns data, synthesize it into a direct answer.",
		),
	}

	for _, m := range conv.Messages {
		switch m.Role {
		case model.RoleUser:
			msgs = append(msgs, openai.UserMessage(m.Content))
		case model.RoleAssistant:
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}

	// Build a lookup map and the OpenAI tool param slice from the registry.
	toolMap := make(map[string]Tool, len(a.tools))
	toolDefs := make([]openai.ChatCompletionToolUnionParam, len(a.tools))
	for i, t := range a.tools {
		toolMap[t.Name()] = t
		toolDefs[i] = t.Definition()
	}

	for i := 0; i < 15; i++ {
		resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4_1,
			Messages: msgs,
			Tools:    toolDefs,
		})

		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		message := resp.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			return message.Content, nil
		}

		msgs = append(msgs, message.ToParam())

		for _, call := range message.ToolCalls {
			slog.InfoContext(ctx, "Tool call received", "name", call.Function.Name, "args", call.Function.Arguments)

			t, ok := toolMap[call.Function.Name]
			if !ok {
				return "", errors.New("unknown tool call: " + call.Function.Name)
			}

			result, err := t.Execute(ctx, call.Function.Arguments)
			if err != nil {
				return "", err
			}

			msgs = append(msgs, openai.ToolMessage(result, call.ID))
		}
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}
