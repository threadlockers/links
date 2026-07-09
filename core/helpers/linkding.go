package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type LinkdingConfig struct {
	BaseApiUrl string
	ApiToken   string
}

type BookmarkResult struct {
	ID       int      `json:"id"`
	URL      string   `json:"url"`
	TagNames []string `json:"tag_names"`
}

type BookmarkListResponse struct {
	Count   int              `json:"count"`
	Results []BookmarkResult `json:"results"`
}

var MAX_DESCRIPTION_LENGTH = 200

func AddBookmarkToLinkding(cfg LinkdingConfig, url, title, description, poster, remainingText string) error {
	notes := fmt.Sprintf("Posted by: @%s", poster)
	if remainingText != "" {
		notes += fmt.Sprintf("\nAdditional description: %s", remainingText)
	}

	if len(description) > MAX_DESCRIPTION_LENGTH {
		description = description[:MAX_DESCRIPTION_LENGTH] + "..."
	}

	payload := map[string]any{
		"url":         url,
		"title":       title,
		"description": description,
		"notes":       notes,
		"shared":      true,
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/bookmarks/", cfg.BaseApiUrl), bytes.NewBuffer(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Token %s", cfg.ApiToken))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.New(string(body))
	}

	return nil
}

// returns the first match
func GetBookmarkByUrl(cfg LinkdingConfig, url string) (*BookmarkResult, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/bookmarks/?q=%s", cfg.BaseApiUrl, url), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Token %s", cfg.ApiToken))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result BookmarkListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode bookmark search response: %w", err)
	}

	if result.Count == 0 {
		return nil, nil
	}

	return &result.Results[0], nil
}

// UpdateBookmarkTags replaces all tags on a bookmark with the given tag list.
// Callers should merge existing tags before calling this.
func UpdateBookmarkTags(cfg LinkdingConfig, id int, tags []string) error {
	payload := map[string]any{
		"tag_names": tags,
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/bookmarks/%d/", cfg.BaseApiUrl, id), bytes.NewBuffer(jsonBody))
	req.Header.Add("Authorization", fmt.Sprintf("Token %s", cfg.ApiToken))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update bookmark tags (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
