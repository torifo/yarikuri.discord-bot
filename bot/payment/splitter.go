package payment

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"

	"github.com/yarikuri/models"
)

// CreateSplitAmountEmbed は金額分割処理用のEmbedメッセージを作成
func CreateSplitAmountEmbed(originalData *models.ConfirmationData, totalAmount, remainingAmount int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "📊 金額分割処理",
		Color: 0xff9900,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "✅ 保存完了", Value: fmt.Sprintf("¥%d", originalData.Amount), Inline: true},
			{Name: "📋 総額", Value: fmt.Sprintf("¥%d", totalAmount), Inline: true},
			{Name: "💰 残額", Value: fmt.Sprintf("¥%d", remainingAmount), Inline: true},
		},
		Description: "総額より少ない金額が入力されました。残りの金額分のエントリを作成してください。",
		Footer: &discordgo.MessageEmbedFooter{
			Text: "下記で残額分のカテゴリーや詳細を設定してください。",
		},
	}
}

// CreateSplitAmountComponents は金額分割処理用のコンポーネントを作成
func CreateSplitAmountComponents(remainingAmount int, messageID string, state *models.BotState) []discordgo.MessageComponent {
	// カテゴリー選択用のSelectMenuオプションを準備（最初の25件）
	var categoryOptions []discordgo.SelectMenuOption
	categories := state.GetMasterData("categories").([]models.Category)
	for _, category := range categories {
		if len(categoryOptions) >= 25 {
			break
		}
		categoryOptions = append(categoryOptions, discordgo.SelectMenuOption{
			Label: category.Name,
			Value: strconv.Itoa(category.ID),
		})
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "split_category_select:" + messageID,
					Placeholder: "残額分のカテゴリーを選択...",
					Options:     categoryOptions,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: "split_detail_edit:" + messageID,
					Label:    "詳細を編集",
					Style:    discordgo.SecondaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
				},
				discordgo.Button{
					CustomID: "split_save:" + messageID,
					Label:    fmt.Sprintf("残額 ¥%d で保存", remainingAmount),
					Style:    discordgo.PrimaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "💾"},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: "split_cancel:" + messageID,
					Label:    "分割をやめる",
					Style:    discordgo.DangerButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
			},
		},
	}
}

// ProcessAmountSplit は金額分割処理のロジックを実行
func ProcessAmountSplit(originalData *models.ConfirmationData, inputAmount int) (totalAmount, remainingAmount int, shouldSplit bool) {
	// 総額を判定（AI解析結果の総額または元の金額を使用）
	totalAmount = originalData.Amount
	if originalData.AIResult.TotalAmount != nil && *originalData.AIResult.TotalAmount > 0 {
		totalAmount = *originalData.AIResult.TotalAmount
	}

	// 入力金額が総額より小さい場合は分割処理
	if inputAmount < totalAmount {
		remainingAmount = totalAmount - inputAmount
		shouldSplit = true
		return totalAmount, remainingAmount, shouldSplit
	}

	// 分割不要
	return totalAmount, 0, false
}

// CreateSplitConfirmationData は分割用の確認データを作成
func CreateSplitConfirmationData(originalData *models.ConfirmationData, remainingAmount int, messageID string) *models.ConfirmationData {
	return &models.ConfirmationData{
		MessageID:        messageID,
		Date:             originalData.Date,
		Amount:           remainingAmount,
		CategoryID:       originalData.CategoryID, // 元のカテゴリーをデフォルトとして設定
		GroupID:          originalData.GroupID,
		UserID:           originalData.UserID,
		Detail:           "分割エントリ（残額分）",
		PaymentMethod:    originalData.PaymentMethod,
		AIResult:         originalData.AIResult,
		OriginalAmount:   &originalData.Amount,
		RemainingAmount:  &remainingAmount,
		IsPartialEntry:   true,
		ParentMessageID:  &originalData.MessageID,
	}
}