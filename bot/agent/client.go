package agent

import (
	"context"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/yarikuri/errors"
)

// Client はGemini AIクライアントを管理
type Client struct {
	model *genai.GenerativeModel
}

// NewClient は新しいAIクライアントを作成
func NewClient(apiKey string) (*Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, errors.NewBotError(errors.ErrorTypeAIService, "Gemini APIクライアントの初期化に失敗", err).
			WithContext("api_key_set", apiKey != "")
	}

	return &Client{
		model: client.GenerativeModel("gemini-1.5-flash-latest"),
	}, nil
}

// GetModel はGenerativeModelを返す
func (c *Client) GetModel() *genai.GenerativeModel {
	return c.model
}