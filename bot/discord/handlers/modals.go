package handlers

import (
	"github.com/bwmarrin/discordgo"
)

// HandleAmountInputModal は金額入力モーダルを処理
func HandleAmountInputModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "金額入力モーダル処理は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleEditAmountModal は金額編集モーダルを処理
func HandleEditAmountModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "金額編集モーダル処理は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleEditDetailModal は詳細編集モーダルを処理
func HandleEditDetailModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "詳細編集モーダル処理は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleEditPaymentModal は支払い方法編集モーダルを処理
func HandleEditPaymentModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "支払い方法編集モーダル処理は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleEditDateModal は日付編集モーダルを処理
func HandleEditDateModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "日付編集モーダル処理は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleAddCategoryModal はカテゴリ追加モーダルを処理
func HandleAddCategoryModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "カテゴリ追加モーダル処理は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleAddGroupModal はグループ追加モーダルを処理
func HandleAddGroupModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "グループ追加モーダル処理は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleAddUserModal はユーザ追加モーダルを処理
func HandleAddUserModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "ユーザ追加モーダル処理は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}