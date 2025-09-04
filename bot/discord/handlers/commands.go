package handlers

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// HandleCheckMaster はcheck_masterコマンドを処理
func HandleCheckMaster(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 実装は後で追加
	embed := &discordgo.MessageEmbed{
		Title: "マスターデータ読み込み状況",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Status", Value: "実装中", Inline: true},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

// HandleShowMaster はshow_masterコマンドを処理
func HandleShowMaster(s *discordgo.Session, i *discordgo.InteractionCreate) {
	dataType := i.ApplicationCommandData().Options[0].StringValue()
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("「%s」データの表示機能は実装中です", dataType),
		},
	})
}

// HandleAdd はaddコマンドを処理
func HandleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "手動追加機能は実装中です",
		},
	})
}

// HandleFix はfixコマンドを処理
func HandleFix(s *discordgo.Session, i *discordgo.InteractionCreate) {
	keyword := i.ApplicationCommandData().Options[0].StringValue()
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("「%s」の修正機能は実装中です", keyword),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleAddMaster はadd_masterコマンドを処理
func HandleAddMaster(s *discordgo.Session, i *discordgo.InteractionCreate) {
	masterType := i.ApplicationCommandData().Options[0].StringValue()
	name := i.ApplicationCommandData().Options[1].StringValue()
	
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("「%s」に「%s」を追加する機能は実装中です", masterType, name),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}