package discord

import "github.com/bwmarrin/discordgo"

// GetApplicationCommands はBotで使用するDiscordコマンドを返す
func GetApplicationCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "check_master",
			Description: "メモリに読み込まれているマスターデータの件数を確認します。",
		},
		{
			Name:        "show_master",
			Description: "指定したマスターデータのリストを表示します。",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "表示したいマスターデータの種類",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "カテゴリ", Value: "category"},
						{Name: "グループ", Value: "group"},
						{Name: "ユーザー", Value: "user"},
						{Name: "支払い方法", Value: "payment_type"},
					},
				},
			},
		},
		{
			Name:        "add",
			Description: "レシートがない支出を手動で追加します。",
		},
		{
			Name:        "fix",
			Description: "キューに追加された未同期のデータを修正します。",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "keyword",
					Description: "修正したいデータのキーワード",
					Required:    true,
				},
			},
		},
		{
			Name:        "add_master",
			Description: "新しいマスターデータ（カテゴリ、グループ、ユーザー、支払い方法）をキューに追加します。",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "追加するマスターデータの種類",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "カテゴリ", Value: "category"},
						{Name: "グループ", Value: "group"},
						{Name: "ユーザー", Value: "user"},
						{Name: "支払い方法", Value: "payment_type"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "追加するデータの名前",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type_name",
					Description: "支払い方法の場合のみ：種別名（例：カード、電子マネー）",
					Required:    false,
				},
			},
		},
	}
}

// GetCommandHandlers はコマンド名とハンドラ関数のマップを返す
func GetCommandHandlers() map[string]func(*discordgo.Session, *discordgo.InteractionCreate) {
	return map[string]func(*discordgo.Session, *discordgo.InteractionCreate){
		"check_master": HandleCheckMaster,
		"show_master":  HandleShowMaster,
		"add":          HandleAdd,
		"fix":          HandleFix,
		"add_master":   HandleAddMaster,
	}
}

// GetComponentHandlers はカスタムIDプレフィックスとハンドラ関数のマップを返す
func GetComponentHandlers() map[string]func(*discordgo.Session, *discordgo.InteractionCreate) {
	return map[string]func(*discordgo.Session, *discordgo.InteractionCreate){
		"pagination":               HandlePagination,
		"receipt_info_button":      HandleReceiptInfoButton,
		"category_select":          HandleCategorySelect,
		"group_select":             HandleGroupSelect,
		"user_select":              HandleUserSelect,
		"add_category_modal":       HandleAddCategoryModal,
		"add_group_modal":          HandleAddGroupModal,
		"add_user_modal":           HandleAddUserModal,
		"save_data":                HandleSaveData,
		"confirm_save":             HandleConfirmSave,
		"edit_amount":              HandleEditAmount,
		"edit_category":            HandleEditCategory,
		"edit_group":               HandleEditGroup,
		"edit_user":                HandleEditUser,
		"edit_detail":              HandleEditDetail,
		"edit_payment":             HandleEditPayment,
		"edit_date":                HandleEditDate,
		"cancel_entry":             HandleCancelEntry,
		"amount_input_modal":       HandleAmountInputModal,
		"edit_amount_modal":        HandleEditAmountModal,
		"edit_detail_modal":        HandleEditDetailModal,
		"edit_payment_modal":       HandleEditPaymentModal,
		"edit_date_modal":          HandleEditDateModal,
		"payment_method_select":    HandlePaymentMethodSelect,
		"credit_detail_select":     HandleCreditDetailSelect,
		"split_category_select":    HandleSplitCategorySelect,
		"split_detail_edit":        HandleSplitDetailEdit,
		"split_save":               HandleSplitSave,
		"split_cancel":             HandleSplitCancel,
	}
}