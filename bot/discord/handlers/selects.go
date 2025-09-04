package handlers

import (
	"github.com/bwmarrin/discordgo"
)

// HandleCategorySelect はカテゴリ選択メニューを処理
func HandleCategorySelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "カテゴリ選択機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleGroupSelect はグループ選択メニューを処理
func HandleGroupSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "グループ選択機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleUserSelect はユーザ選択メニューを処理
func HandleUserSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "ユーザ選択機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandlePaymentMethodSelect は支払い方法選択メニューを処理
func HandlePaymentMethodSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "支払い方法選択機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleCreditDetailSelect はクレジット詳細選択メニューを処理
func HandleCreditDetailSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "クレジット詳細選択機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleSplitCategorySelect は分割カテゴリ選択メニューを処理
func HandleSplitCategorySelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "分割カテゴリ選択機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}