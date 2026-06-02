package assistant

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go/v2"
)

// weatherTool returns current weather conditions for a given location.
type weatherTool struct{}

func (weatherTool) Name() string { return "get_weather" }

func (weatherTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "get_weather",
		Description: openai.String("Get the current weather at the given location"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]string{"type": "string"},
			},
			"required": []string{"location"},
		},
	})
}

func (weatherTool) Execute(ctx context.Context, args string) (string, error) {
	var payload struct {
		Location string `json:"location"`
	}
	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return "failed to parse tool call arguments: " + err.Error(), nil
	}
	result, err := GetWeather(ctx, payload.Location)
	if err != nil {
		return "failed to get weather", nil
	}
	return result, nil
}

// forecastTool returns a multi-day weather forecast for a given location.
type forecastTool struct{}

func (forecastTool) Name() string { return "get_forecast" }

func (forecastTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "get_forecast",
		Description: openai.String("Get a multi-day weather forecast (up to 3 days) for the given location"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]string{"type": "string"},
				"days": map[string]string{
					"type":        "integer",
					"description": "Number of days to forecast, between 1 and 3. Defaults to 3 if not provided.",
				},
			},
			"required": []string{"location"},
		},
	})
}

func (forecastTool) Execute(ctx context.Context, args string) (string, error) {
	var payload struct {
		Location string `json:"location"`
		Days     int    `json:"days"`
	}
	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return "failed to parse tool call arguments: " + err.Error(), nil
	}
	result, err := GetForecast(ctx, payload.Location, payload.Days)
	if err != nil {
		return "failed to get forecast", nil
	}
	return result, nil
}
