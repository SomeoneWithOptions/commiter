package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const openRouterEndpoint = "https://openrouter.ai/api/v1/chat/completions"

func generateCommitMessage(diff string, model string) (string, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY environment variable is not set")
	}

	systemPrompt := getSystemPrompt()

	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": diff},
		},
		"temperature": 0.3,
		"max_tokens":  2000,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", openRouterEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w (body: %s)", err, string(bodyBytes))
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("API returned empty choices")
	}

	message := cleanCommitMessage(response.Choices[0].Message.Content)
	if message == "" {
		return "", fmt.Errorf("API returned empty commit message - raw response: %s", string(bodyBytes))
	}

	return message, nil
}

func cleanCommitMessage(msg string) string {
	// Trim whitespace
	msg = strings.TrimSpace(msg)

	// To lower case
	msg = strings.ToLower(msg)

	// Remove trailing dot
	msg = strings.TrimSuffix(msg, ".")

	// Remove common conventional commit prefixes if present (e.g., "feat: ", "fix(ui): ")
	// This regex matches "word", optionally "word(scope)", followed by ": "
	re := regexp.MustCompile(`^[a-z]+(\([a-z0-9-]+\))?:\s+`)
	msg = re.ReplaceAllString(msg, "")

	// Normalize spaces (replace multiple spaces with single space)
	reSpaces := regexp.MustCompile(`\s+`)
	msg = reSpaces.ReplaceAllString(msg, " ")

	return strings.TrimSpace(msg)
}
