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

func AddBookmarkToLinkding(cfg LinkdingConfig, url, title, description, poster string) error {
	payload := map[string]any{
		"url":         url,
		"title":       title,
		"description": description,
		"notes":       fmt.Sprintf("Posted by: @%s", poster),
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
