package assistant

import (
	"context"
	"time"

	"github.com/openai/openai-go/v2"
)

// dateTool returns the current date and time in RFC3339 format.
type dateTool struct{}

func (dateTool) Name() string { return "get_today_date" }

func (dateTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "get_today_date",
		Description: openai.String("Get today's date and time in RFC3339 format"),
	})
}

func (dateTool) Execute(_ context.Context, _ string) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}
