package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DeepseekProvider struct {
	config ModelConfig
	client *http.Client
}

type DeepseekRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	Tools       []DeepseekTool `json:"tools,omitempty"`
}

type DeepseekTool struct {
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

type DeepseekResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func NewDeepseekProvider(config ModelConfig) *DeepseekProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com"
	}
	if config.Model == "" {
		config.Model = "deepseek-chat"
	}

	return &DeepseekProvider{
		config: config,
		client: &http.Client{},
	}
}

func (d *DeepseekProvider) Chat(ctx context.Context, messages []Message, tools []map[string]interface{}) (string, []ToolCall, error) {
	reqBody := DeepseekRequest{
		Model:    d.config.Model,
		Messages: messages,
	}
	
	// 添加工具定义
	if len(tools) > 0 {
		deepseekTools := make([]DeepseekTool, len(tools))
		for i, tool := range tools {
			deepseekTools[i] = DeepseekTool{
				Type:     tool["type"].(string),
				Function: tool["function"].(map[string]interface{}),
			}
		}
		reqBody.Tools = deepseekTools
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.config.APIKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deepseekResp DeepseekResponse
	if err := json.Unmarshal(body, &deepseekResp); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(deepseekResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in response")
	}

	choice := deepseekResp.Choices[0]
	return choice.Message.Content, choice.Message.ToolCalls, nil
}

func (d *DeepseekProvider) GetModels(ctx context.Context) ([]string, error) {
	return []string{"deepseek-chat", "deepseek-coder"}, nil
}