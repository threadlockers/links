package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

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

	re := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	match := re.FindSubmatch(body)
	if match == nil {
		return "", fmt.Errorf("no title found")
	}

	title := strings.TrimSpace(string(match[1]))
	return title, nil
}
