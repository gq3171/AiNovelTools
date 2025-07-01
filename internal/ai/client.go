package ai

import (
	"context"
	"fmt"
)

type Provider string

const (
	ProviderZhipu    Provider = "zhipu"
	ProviderDeepseek Provider = "deepseek"
)

type Config struct {
	Provider   Provider          `yaml:"provider"`
	Models     map[Provider]ModelConfig `yaml:"models"`
	MaxTokens  int              `yaml:"max_tokens"`
	Temperature float64         `yaml:"temperature"`
}

type ModelConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

type Client struct {
	config Config
	provider AIProvider
}

type AIProvider interface {
	Chat(ctx context.Context, messages []Message) (string, []ToolCall, error)
	GetModels(ctx context.Context) ([]string, error)
}

func NewClient(config Config) *Client {
	var provider AIProvider
	
	switch config.Provider {
	case ProviderZhipu:
		provider = NewZhipuProvider(config.Models[ProviderZhipu])
	case ProviderDeepseek:
		provider = NewDeepseekProvider(config.Models[ProviderDeepseek])
	default:
		provider = NewZhipuProvider(config.Models[ProviderZhipu])
	}

	return &Client{
		config:   config,
		provider: provider,
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, []ToolCall, error) {
	return c.provider.Chat(ctx, messages)
}

func (c *Client) GetModels(ctx context.Context) ([]string, error) {
	return c.provider.GetModels(ctx)
}

func (c *Client) SwitchProvider(provider Provider) error {
	modelConfig, exists := c.config.Models[provider]
	if !exists {
		return fmt.Errorf("no configuration found for provider: %s", provider)
	}
	
	if modelConfig.APIKey == "" {
		return fmt.Errorf("no API key configured for provider: %s", provider)
	}
	
	c.config.Provider = provider
	
	switch provider {
	case ProviderZhipu:
		c.provider = NewZhipuProvider(modelConfig)
	case ProviderDeepseek:
		c.provider = NewDeepseekProvider(modelConfig)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	
	return nil
}