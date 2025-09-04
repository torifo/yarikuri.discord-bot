package handlers

import (
	"github.com/bwmarrin/discordgo"
)

// HandleReceiptInfoButton はレシート情報入力ボタンを処理
func HandleReceiptInfoButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "レシート情報入力機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleSaveData はデータ保存ボタンを処理
func HandleSaveData(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "データ保存機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleConfirmSave は確認保存ボタンを処理
func HandleConfirmSave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "確認保存機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleCancelEntry はエントリキャンセルボタンを処理
func HandleCancelEntry(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "エントリをキャンセルしました",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandlePagination はページネーションボタンを処理
func HandlePagination(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "ページネーション機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// 編集ボタン群のスタブ
func HandleEditAmount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "金額編集機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleEditCategory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "カテゴリ編集機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleEditGroup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "グループ編集機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleEditUser(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "ユーザ編集機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleEditDetail(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "詳細編集機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleEditPayment(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "支払い方法編集機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleEditDate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "日付編集機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// 分割関連ボタン群のスタブ
func HandleSplitDetailEdit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "分割詳細編集機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleSplitSave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "分割保存機能は実装中です",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func HandleSplitCancel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "分割をキャンセルしました",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}