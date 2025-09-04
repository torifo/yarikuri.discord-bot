package ui

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"

	"github.com/yarikuri/models"
)

// CreateCategorySelectMenu はカテゴリ選択メニューを作成
func CreateCategorySelectMenu(categories []models.Category, customID, placeholder string) discordgo.SelectMenu {
	var options []discordgo.SelectMenuOption
	for _, category := range categories {
		if len(options) >= 25 { // Discord SelectMenuの制限
			break
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: category.Name,
			Value: strconv.Itoa(category.ID),
		})
	}

	return discordgo.SelectMenu{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// CreateGroupSelectMenu はグループ選択メニューを作成
func CreateGroupSelectMenu(groups []models.Group, customID, placeholder string) discordgo.SelectMenu {
	var options []discordgo.SelectMenuOption
	for _, group := range groups {
		if len(options) >= 25 {
			break
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: group.Name,
			Value: strconv.Itoa(group.ID),
		})
	}

	return discordgo.SelectMenu{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// CreateUserSelectMenu はユーザ選択メニューを作成
func CreateUserSelectMenu(users []models.User, customID, placeholder string) discordgo.SelectMenu {
	var options []discordgo.SelectMenuOption
	for _, user := range users {
		if len(options) >= 25 {
			break
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: user.Name,
			Value: strconv.Itoa(user.ID),
		})
	}

	return discordgo.SelectMenu{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// CreatePaymentMethodSelectMenu は支払い方法選択メニューを作成
func CreatePaymentMethodSelectMenu(paymentTypes []models.PaymentType, customID, placeholder string) discordgo.SelectMenu {
	var options []discordgo.SelectMenuOption
	for _, paymentType := range paymentTypes {
		if len(options) >= 25 {
			break
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: paymentType.PayKind,
			Value: strconv.Itoa(paymentType.PayID),
		})
	}

	return discordgo.SelectMenu{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// CreateConfirmationButtons は確認画面用のボタン群を作成
func CreateConfirmationButtons(messageID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: "save_data:" + messageID,
					Label:    "保存",
					Style:    discordgo.PrimaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "💾"},
				},
				discordgo.Button{
					CustomID: "cancel_entry:" + messageID,
					Label:    "キャンセル",
					Style:    discordgo.DangerButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
			},
		},
	}
}

// CreateEditButtons は編集用のボタン群を作成
func CreateEditButtons(messageID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: "edit_amount:" + messageID,
					Label:    "金額",
					Style:    discordgo.SecondaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "💰"},
				},
				discordgo.Button{
					CustomID: "edit_category:" + messageID,
					Label:    "カテゴリ",
					Style:    discordgo.SecondaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "📂"},
				},
				discordgo.Button{
					CustomID: "edit_group:" + messageID,
					Label:    "グループ",
					Style:    discordgo.SecondaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "👥"},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: "edit_user:" + messageID,
					Label:    "ユーザ",
					Style:    discordgo.SecondaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "👤"},
				},
				discordgo.Button{
					CustomID: "edit_detail:" + messageID,
					Label:    "詳細",
					Style:    discordgo.SecondaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
				},
				discordgo.Button{
					CustomID: "edit_payment:" + messageID,
					Label:    "支払方法",
					Style:    discordgo.SecondaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "💳"},
				},
			},
		},
	}
}

// CreatePaginationButtons はページネーション用のボタン群を作成
func CreatePaginationButtons(dataType string, currentPage, totalPages int) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent

	if currentPage > 0 || currentPage < totalPages-1 {
		var buttons []discordgo.MessageComponent

		if currentPage > 0 {
			buttons = append(buttons, discordgo.Button{
				CustomID: fmt.Sprintf("pagination:%s:%d", dataType, currentPage-1),
				Label:    "前のページ",
				Style:    discordgo.SecondaryButton,
				Emoji:    &discordgo.ComponentEmoji{Name: "⬅️"},
			})
		}

		if currentPage < totalPages-1 {
			buttons = append(buttons, discordgo.Button{
				CustomID: fmt.Sprintf("pagination:%s:%d", dataType, currentPage+1),
				Label:    "次のページ",
				Style:    discordgo.SecondaryButton,
				Emoji:    &discordgo.ComponentEmoji{Name: "➡️"},
			})
		}

		components = append(components, discordgo.ActionsRow{Components: buttons})
	}

	return components
}