package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ZhipuProvider struct {
	config ModelConfig
	client *http.Client
}

type ZhipuRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	Tools       []ZhipuTool    `json:"tools,omitempty"`
}

type ZhipuTool struct {
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

type ZhipuResponse struct {
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

func NewZhipuProvider(config ModelConfig) *ZhipuProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://open.bigmodel.cn/api/paas/v4"
	}
	if config.Model == "" {
		config.Model = "glm-4"
	}

	return &ZhipuProvider{
		config: config,
		client: &http.Client{},
	}
}

func (z *ZhipuProvider) Chat(ctx context.Context, messages []Message) (string, []ToolCall, error) {
	reqBody := ZhipuRequest{
		Model:    z.config.Model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", z.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+z.config.APIKey)

	resp, err := z.client.Do(req)
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

	var zhipuResp ZhipuResponse
	if err := json.Unmarshal(body, &zhipuResp); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(zhipuResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in response")
	}

	choice := zhipuResp.Choices[0]
	return choice.Message.Content, choice.Message.ToolCalls, nil
}

func (z *ZhipuProvider) GetModels(ctx context.Context) ([]string, error) {
	return []string{"glm-4", "glm-4-flash", "glm-4-plus"}, nil
}