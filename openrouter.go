package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const openRouterEndpoint = "https://openrouter.ai/api/v1/chat/completions"
const maxDiffPromptChars = 30000

func generateCommitMessage(diff string, model string) (string, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY environment variable is not set")
	}

	systemPrompt := getSystemPrompt()
	diffForPrompt := limitDiffForPrompt(diff, maxDiffPromptChars)

	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": diffForPrompt},
		},
		"temperature":       0.3,
		"max_tokens":        3000,
		"include_reasoning": false,
		"reasoning_effort":  "low",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 20 * time.Second,
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
			FinishReason *string `json:"finish_reason"`
			Message      struct {
				Content   json.RawMessage `json:"content"`
				Reasoning string          `json:"reasoning"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w (body: %s)", err, string(bodyBytes))
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("API returned empty choices")
	}

	content, err := extractMessageContent(response.Choices[0].Message.Content)
	if err != nil {
		return "", fmt.Errorf("failed to extract message content: %w (body: %s)", err, string(bodyBytes))
	}

	message := cleanCommitMessage(content)
	if message == "" {
		finishReason := "unknown"
		if response.Choices[0].FinishReason != nil && *response.Choices[0].FinishReason != "" {
			finishReason = *response.Choices[0].FinishReason
		}
		return "", fmt.Errorf("API returned empty commit message (finish_reason=%s) - raw response: %s", finishReason, string(bodyBytes))
	}

	return message, nil
}

func extractMessageContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString, nil
	}

	var asParts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &asParts); err == nil {
		var b strings.Builder
		for _, part := range asParts {
			if part.Text == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(part.Text)
		}
		return b.String(), nil
	}

	return "", fmt.Errorf("unsupported content format: %s", string(raw))
}

func limitDiffForPrompt(diff string, maxChars int) string {
	if maxChars <= 0 || len(diff) <= maxChars {
		return diff
	}

	const marker = "\n\n[... diff truncated to fit model context ...]\n\n"
	if maxChars <= len(marker) {
		return diff[:maxChars]
	}
	headLen := (maxChars - len(marker)) / 2
	if headLen < 0 {
		headLen = 0
	}
	tailLen := maxChars - len(marker) - headLen
	if tailLen < 0 {
		tailLen = 0
	}

	head := diff[:headLen]
	tail := diff[len(diff)-tailLen:]
	return head + marker + tail
}

func cleanCommitMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	msg = strings.Join(strings.Fields(msg), " ")
	msg = strings.ToLower(msg)
	msg = strings.TrimSuffix(msg, ".")
	msg = strings.TrimSpace(msg)
	return msg
}
