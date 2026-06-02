package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// weatherBaseURL is the WeatherAPI.com endpoint host. https://www.weatherapi.com/docs/
const weatherBaseURL = "https://api.weatherapi.com/v1"

// weatherHTTPClient is shared across weather calls with a sane timeout.
var weatherHTTPClient = &http.Client{Timeout: 10 * time.Second}

// weatherResponse models the subset of the WeatherAPI.com response we use.
type weatherResponse struct {
	Location struct {
		Name      string `json:"name"`
		Region    string `json:"region"`
		Country   string `json:"country"`
		Localtime string `json:"localtime"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		Condition struct {
			Text string `json:"text"`
		} `json:"condition"`
		WindKph    float64 `json:"wind_kph"`
		Humidity   int     `json:"humidity"`
		FeelslikeC float64 `json:"feelslike_c"`
	} `json:"current"`
	Forecast struct {
		Forecastday []struct {
			Date string `json:"date"`
			Day  struct {
				MaxtempC  float64 `json:"maxtemp_c"`
				MintempC  float64 `json:"mintemp_c"`
				Condition struct {
					Text string `json:"text"`
				} `json:"condition"`
				DailyChanceOfRain int `json:"daily_chance_of_rain"`
			} `json:"day"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

// GetWeather returns a compact, human-readable summary of the current weather
// for the given location. It is intended to be fed back to the assistant model.
func GetWeather(ctx context.Context, location string) (string, error) {
	key := os.Getenv("WEATHER_API_KEY")
	if key == "" {
		return "weather service is not configured", nil
	}

	q := url.Values{}
	q.Set("key", key)
	q.Set("q", location)

	data, err := fetchWeather(ctx, "/current.json", q)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch current weather", "location", location, "error", err)
		return fmt.Sprintf("weather data unavailable for %q", location), nil
	}

	loc := data.Location.Name
	if data.Location.Country != "" {
		loc = loc + ", " + data.Location.Country
	}

	return fmt.Sprintf(
		"%s: %.0f°C (feels like %.0f°C), %s, wind %.0f km/h, humidity %d%%",
		loc,
		data.Current.TempC,
		data.Current.FeelslikeC,
		data.Current.Condition.Text,
		data.Current.WindKph,
		data.Current.Humidity,
	), nil
}

// GetForecast returns a compact, multi-day forecast summary for the given
// location. days is clamped to the WeatherAPI.com free-tier range of 1-3.
func GetForecast(ctx context.Context, location string, days int) (string, error) {
	key := os.Getenv("WEATHER_API_KEY")
	if key == "" {
		return "weather service is not configured", nil
	}

	if days < 1 {
		days = 1
	}
	if days > 3 {
		days = 3
	}

	q := url.Values{}
	q.Set("key", key)
	q.Set("q", location)
	q.Set("days", fmt.Sprintf("%d", days))

	data, err := fetchWeather(ctx, "/forecast.json", q)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch forecast", "location", location, "error", err)
		return fmt.Sprintf("forecast data unavailable for %q", location), nil
	}

	loc := data.Location.Name
	if data.Location.Country != "" {
		loc = loc + ", " + data.Location.Country
	}

	lines := make([]string, 0, len(data.Forecast.Forecastday)+1)
	lines = append(lines, "Forecast for "+loc+":")
	for _, fd := range data.Forecast.Forecastday {
		lines = append(lines, fmt.Sprintf(
			"%s: %.0f–%.0f°C, %s, %d%% chance of rain",
			fd.Date,
			fd.Day.MintempC,
			fd.Day.MaxtempC,
			fd.Day.Condition.Text,
			fd.Day.DailyChanceOfRain,
		))
	}

	return strings.Join(lines, "\n"), nil
}

// fetchWeather performs the HTTP GET against WeatherAPI.com and decodes the response.
func fetchWeather(ctx context.Context, path string, q url.Values) (*weatherResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, weatherBaseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := weatherHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	var out weatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
