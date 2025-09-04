package discord

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/yarikuri/errors"
	"github.com/yarikuri/models"
)

// SessionManager はDiscordセッションを管理する
type SessionManager struct {
	session *discordgo.Session
	state   *models.BotState
}

// NewSessionManager は新しいセッションマネージャを作成
func NewSessionManager(botToken string, state *models.BotState) (*SessionManager, error) {
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, errors.NewBotError(errors.ErrorTypeDiscordAPI, "Discordセッション作成エラー", err).
			WithContext("bot_token_set", botToken != "")
	}

	sm := &SessionManager{
		session: dg,
		state:   state,
	}

	// イベントハンドラを設定
	sm.setupEventHandlers()

	return sm, nil
}

// setupEventHandlers はイベントハンドラを設定
func (sm *SessionManager) setupEventHandlers() {
	sm.session.AddHandler(sm.messageCreate)
	sm.session.AddHandler(sm.interactionCreate)
}

// Start はセッションを開始
func (sm *SessionManager) Start() error {
	err := sm.session.Open()
	if err != nil {
		return errors.NewBotError(errors.ErrorTypeDiscordAPI, "Discord接続開始エラー", err)
	}

	// アプリケーションコマンドを登録
	commands := GetApplicationCommands()
	for _, command := range commands {
		_, err := sm.session.ApplicationCommandCreate(sm.session.State.User.ID, "", command)
		if err != nil {
			log.Printf("コマンド登録エラー '%s': %v", command.Name, err)
		}
	}

	log.Println("Discordセッションが開始されました。")
	return nil
}

// Close はセッションを終了
func (sm *SessionManager) Close() error {
	return sm.session.Close()
}

// messageCreate はメッセージ作成イベントを処理
func (sm *SessionManager) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Botのメッセージは無視
	if m.Author.ID == s.State.User.ID {
		return
	}

	// 指定されたチャンネル以外は無視
	if m.ChannelID != sm.state.TargetChannelID {
		return
	}

	// 添付ファイル（画像）があるメッセージを処理
	if len(m.Attachments) > 0 {
		HandleReceiptMessage(s, m, sm.state)
	}
}

// interactionCreate はインタラクション作成イベントを処理
func (sm *SessionManager) interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		sm.handleApplicationCommand(s, i)
	case discordgo.InteractionMessageComponent:
		sm.handleMessageComponent(s, i)
	case discordgo.InteractionModalSubmit:
		sm.handleModalSubmit(s, i)
	}
}

// handleApplicationCommand はアプリケーションコマンドを処理
func (sm *SessionManager) handleApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commandHandlers := GetCommandHandlers()
	if handler, exists := commandHandlers[i.ApplicationCommandData().Name]; exists {
		handler(s, i)
	}
}

// handleMessageComponent はメッセージコンポーネントを処理
func (sm *SessionManager) handleMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	componentHandlers := GetComponentHandlers()

	// カスタムIDのプレフィックスでハンドラを検索
	for prefix, handler := range componentHandlers {
		if strings.HasPrefix(customID, prefix) {
			handler(s, i)
			return
		}
	}

	log.Printf("未処理のコンポーネント: %s", customID)
}

// handleModalSubmit はモーダル送信を処理
func (sm *SessionManager) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	componentHandlers := GetComponentHandlers()

	// カスタムIDのプレフィックスでハンドラを検索
	for prefix, handler := range componentHandlers {
		if strings.HasPrefix(customID, prefix) {
			handler(s, i)
			return
		}
	}

	log.Printf("未処理のモーダル: %s", customID)
}