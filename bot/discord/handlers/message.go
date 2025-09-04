package handlers

import (
	"github.com/bwmarrin/discordgo"

	"github.com/yarikuri/agent"
	"github.com/yarikuri/config"
	"github.com/yarikuri/models"
)

// HandleReceiptMessage はレシート画像付きメッセージを処理
func HandleReceiptMessage(s *discordgo.Session, m *discordgo.MessageCreate, state *models.BotState, aiClient *agent.Client, constants config.Constants) {
	// トランザクション状態を作成
	state.SetTransaction(m.ID, &models.TransactionState{
		InitialMessageID: m.ID,
		ImagePath:        "",
		UserInput:        make(map[string]string),
		AIResultChan:     make(chan models.ReceiptAnalysis),
	})

	transaction, _ := state.GetTransaction(m.ID)

	// バックグラウンドでAI解析を開始
	go agent.AnalyzeReceiptInBackground(aiClient, m, transaction, constants.TempImageDir)

	// フォアグラウンドでユーザーに補足情報入力を求めるボタンを表示
	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content: "📋 レシートを解析中です...\n下のボタンをクリックして詳細情報を入力してください:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: "receipt_info_button:" + m.ID,
						Label:    "詳細情報を入力",
						Style:    discordgo.PrimaryButton,
						Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
					},
				},
			},
		},
	})
	if err != nil {
		// エラーログ
	}
}