package assistant

import (
	"context"

	"github.com/openai/openai-go/v2"
)

// Tool is a capability that can be offered to the OpenAI model as a function
// call. Each tool is self-contained: it describes itself and knows how to
// execute itself given the raw JSON arguments the model produces.
type Tool interface {
	// Name returns the function name exposed to the model. It must match the
	// name in Definition().
	Name() string

	// Definition returns the OpenAI tool parameter used when constructing the
	// chat completion request.
	Definition() openai.ChatCompletionToolUnionParam

	// Execute is called when the model invokes this tool. args is the raw JSON
	// string produced by the model. The returned string is sent back to the
	// model as the tool result.
	Execute(ctx context.Context, args string) (string, error)
}
