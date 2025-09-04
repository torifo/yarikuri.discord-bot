package errors

import (
	"encoding/json"
	"fmt"
	"log"
)

// BotError 統一エラー構造体
type BotError struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Cause   error                  `json:"cause,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ErrorType エラー分類
type ErrorType string

const (
	ErrorTypeDiscordAPI     ErrorType = "discord_api"
	ErrorTypeAIService      ErrorType = "ai_service"
	ErrorTypeDataAccess     ErrorType = "data_access"
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeConfiguration ErrorType = "configuration"
	ErrorTypeFileIO         ErrorType = "file_io"
	ErrorTypeNetwork        ErrorType = "network"
)

// NewBotError BotError生成関数
func NewBotError(errorType ErrorType, message string, cause error) *BotError {
	return &BotError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext コンテキスト情報追加
func (e *BotError) WithContext(key string, value interface{}) *BotError {
	e.Context[key] = value
	return e
}

// Error error インターフェース実装
func (e *BotError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// LogBotError 統一ログ出力関数
func LogBotError(err *BotError) {
	contextStr := ""
	if len(err.Context) > 0 {
		contextData, _ := json.Marshal(err.Context)
		contextStr = fmt.Sprintf(" context=%s", string(contextData))
	}

	log.Printf("BotError [%s] %s%s", err.Type, err.Message, contextStr)
	if err.Cause != nil {
		log.Printf("  -> Cause: %v", err.Cause)
	}
}

// HandleError 統一エラーハンドリング関数
func HandleError(err error, fallback func()) {
	if err == nil {
		return
	}

	if botErr, ok := err.(*BotError); ok {
		LogBotError(botErr)
	} else {
		log.Printf("Unexpected error: %v", err)
	}

	if fallback != nil {
		fallback()
	}
}