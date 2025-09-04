package config

import (
	"fmt"
	"os"
)

// BotConfig は Bot の設定を管理する構造体
type BotConfig struct {
	DiscordToken    string
	TargetChannelID string
	GeminiAPIKey    string
}

// Constants はアプリケーション定数を定義
type Constants struct {
	ItemsPerPage      int
	QueueFilePath     string
	TempImageDir      string
	DetailSamplesDir  string
}

// DefaultConstants はデフォルト設定値を返す
func DefaultConstants() Constants {
	return Constants{
		ItemsPerPage:     15,
		QueueFilePath:    "queue.json",
		TempImageDir:     "./bot/img",
		DetailSamplesDir: "./detail_samples",
	}
}

// LoadConfig は環境変数から設定を読み込む
func LoadConfig() (*BotConfig, error) {
	config := &BotConfig{
		DiscordToken:    os.Getenv("TOKEN"),
		TargetChannelID: os.Getenv("CHANNEL_ID"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
	}

	// 必須設定のバリデーション
	if config.DiscordToken == "" {
		return nil, fmt.Errorf("TOKEN environment variable is required")
	}
	if config.TargetChannelID == "" {
		return nil, fmt.Errorf("CHANNEL_ID environment variable is required")
	}
	if config.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	return config, nil
}