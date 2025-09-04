package ui

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/yarikuri/models"
)

// CreateMasterDataEmbed はマスターデータ表示用のEmbedを作成
func CreateMasterDataEmbed(title string, count int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "マスターデータ読み込み状況",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: title, Value: fmt.Sprintf("%d 件", count), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// CreateMasterDataListEmbed はマスターデータリスト表示用のEmbedを作成
func CreateMasterDataListEmbed(dataType string, items []string, currentPage, totalPages int) *discordgo.MessageEmbed {
	description := ""
	for i, item := range items {
		description += fmt.Sprintf("%d. %s\n", i+1, item)
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s リスト", dataType),
		Description: description,
		Color:       0x0099ff,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("ページ %d/%d", currentPage+1, totalPages),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// CreateConfirmationEmbed は確認画面用のEmbedを作成
func CreateConfirmationEmbed(data *models.ConfirmationData) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "📋 支出データ確認",
		Color: 0xffd700,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📅 日付", Value: data.Date, Inline: true},
			{Name: "💰 金額", Value: fmt.Sprintf("¥%d", data.Amount), Inline: true},
			{Name: "📂 カテゴリ", Value: fmt.Sprintf("ID: %d", data.CategoryID), Inline: true},
			{Name: "👤 ユーザ", Value: fmt.Sprintf("ID: %d", data.UserID), Inline: true},
			{Name: "💳 支払方法", Value: data.PaymentMethod, Inline: true},
			{Name: "📝 詳細", Value: data.Detail, Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "内容を確認して保存してください。編集が必要な場合は下のボタンを使用してください。",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// CreateErrorEmbed はエラー表示用のEmbedを作成
func CreateErrorEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xff0000,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// CreateSuccessEmbed は成功表示用のEmbedを作成
func CreateSuccessEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// CreateInfoEmbed は情報表示用のEmbedを作成
func CreateInfoEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x0099ff,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}