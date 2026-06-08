package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

type FxTwitterResponse struct {
	Tweet struct {
		Text   string `json:"text"`
		Author struct {
			Name       string `json:"name"`
			ScreenName string `json:"screen_name"`
		} `json:"author"`
	} `json:"tweet"`
}

func ParseEnv[T any](path string) (T, error) {
	var zero T
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load(path)
		if err != nil {
			return zero, err
		}
	}

	var cfg T
	err := env.Parse(&cfg)
	if err != nil {
		return zero, err
	}

	return cfg, nil
}

func ExtractUrlAndRemainingText(message string) (*url.URL, string) {
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	loc := urlRegex.FindStringIndex(message)
	if loc == nil {
		return nil, message
	}

	rawUrl := message[loc[0]:loc[1]]
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, message
	}

	remaining := message[:loc[0]] + message[loc[1]:]
	remaining = strings.TrimSpace(remaining)
	remaining = strings.Join(strings.Fields(remaining), " ")

	return url, remaining
}

func GetPageTitle(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("returned %d status code: %s", resp.StatusCode, string(body))
	}

	titleElemRegex := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	match := titleElemRegex.FindSubmatch(body)
	if match == nil {
		return "", fmt.Errorf("no title found")
	}

	title := strings.TrimSpace(string(match[1]))
	return title, nil
}

func GetTitleAndDescriptionForTweet(url *url.URL) (string, string, error) {
	originalHost := url.Host
	url.Host = "api.fxtwitter.com"
	resp, err := http.Get(url.String())
	if err != nil {
		return "", "", nil
	}
	defer resp.Body.Close()
	url.Host = originalHost

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("returned %d status code: %s", resp.StatusCode, string(body))
	}

	var data FxTwitterResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", err
	}

	title := fmt.Sprintf("%s (@%s)", data.Tweet.Author.Name, data.Tweet.Author.ScreenName)
	description := data.Tweet.Text

	return title, description, nil
}
