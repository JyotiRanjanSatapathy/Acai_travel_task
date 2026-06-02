package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v2"
)

// itineraryTool generates a day-by-day travel itinerary using a focused
// sub-call to OpenAI. It returns a JSON object of the form:
//
//	{"day1": ["activity1", ...], "day2": [...], ...}
type itineraryTool struct {
	cli openai.Client
}

func (itineraryTool) Name() string { return "generate_itinerary" }

func (itineraryTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "generate_itinerary",
		Description: openai.String("Generate a day-by-day travel itinerary for a given city, number of days, and list of interests"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]string{
					"type":        "string",
					"description": "The city to generate the itinerary for",
				},
				"days": map[string]string{
					"type":        "integer",
					"description": "Number of days for the trip",
				},
				"interest": map[string]any{
					"type":        "array",
					"description": "List of interests to tailor the itinerary (e.g. food, history, art, nature)",
					"items":       map[string]string{"type": "string"},
				},
			},
			"required": []string{"city", "days", "interest"},
		},
	})
}

func (t itineraryTool) Execute(ctx context.Context, args string) (string, error) {
	var payload struct {
		City      string   `json:"city"`
		Days      int      `json:"days"`
		Interest  []string `json:"interest"`
		Interests []string `json:"interests"`
	}
	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return "failed to parse tool call arguments: " + err.Error(), nil
	}

	interests := payload.Interest
	if len(interests) == 0 {
		interests = payload.Interests
	}

	if payload.City == "" || payload.Days < 1 || len(interests) == 0 {
		return "city, days, and interest are required to generate an itinerary", nil
	}

	prompt := fmt.Sprintf(
		"Plan a %d-day trip to %s for someone who likes: %s.",
		payload.Days,
		payload.City,
		strings.Join(interests, ", "),
	)

	resp, err := t.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4_1Mini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(
				`You are a travel planner. Return a JSON object only. ` +
					`Use keys exactly "day1", "day2", up to the requested number of days. ` +
					`Each day value must be an array of 3 to 5 short strings naming specific places or activities. ` +
					`Tailor the plan to the provided interests, avoid duplicates across days, and keep nearby attractions together when reasonable. ` +
					`Do not include explanations, markdown, code fences, or extra keys.`,
			),
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "failed to generate itinerary", nil
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no choices returned from OpenAI for itinerary generation")
	}

	out, err := normalizeItineraryJSON(resp.Choices[0].Message.Content, payload.Days)
	if err != nil {
		return "failed to generate itinerary", nil
	}

	return out, nil
}

func normalizeItineraryJSON(raw string, days int) (string, error) {
	var itinerary map[string][]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &itinerary); err != nil {
		return "", err
	}

	if len(itinerary) == 0 {
		return "", errors.New("empty itinerary response")
	}

	cleaned := make(map[string][]string, days)
	for day := 1; day <= days; day++ {
		key := fmt.Sprintf("day%d", day)
		items, ok := itinerary[key]
		if !ok {
			return "", fmt.Errorf("missing %s", key)
		}

		cleanItems := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			cleanItems = append(cleanItems, item)
		}

		if len(cleanItems) == 0 {
			return "", fmt.Errorf("empty %s itinerary", key)
		}

		cleaned[key] = cleanItems
	}

	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}
