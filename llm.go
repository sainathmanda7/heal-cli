package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMClient struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewLLMClient() (*LLMClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &LLMClient{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *LLMClient) RequestPatchCandidates(ctx context.Context, errorLog, sourceContext string, candidates int) ([]string, error) {
	prompt := fmt.Sprintf(`You are a deterministic backend repair engine.
Analyze this execution failure:
%s

Source context:
%s

Provide ONLY the corrected code to fix the failure. 
CRITICAL GUARDRAILS:
1. Return the COMPLETE, valid, compiling Go file.
2. Output raw code only, no markdown blocks.
3. Handle network/DB failures gracefully (e.g., log and return). DO NOT panic in main().`, errorLog, sourceContext)

	reqPayload := map[string]interface{}{
		"model":       "gpt-4o-mini",
		"messages":    []Message{{Role: "user", Content: prompt}},
		"temperature": 0.4,
		"n":           candidates,
	}

	reqBody, _ := json.Marshal(reqPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("HTTP-Referer", "https://root10x.in")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	var patches []string
	for _, choice := range result.Choices {
		patches = append(patches, strings.TrimSpace(choice.Message.Content))
	}

	return patches, nil
}