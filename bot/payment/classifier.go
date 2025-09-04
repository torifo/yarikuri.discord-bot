package payment

import (
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/yarikuri/models"
)

// EnhancePaymentMethod は支払い方法をより詳細に分類する
func EnhancePaymentMethod(originalPaymentMethod string, state *models.BotState) string {
	paymentLower := strings.ToLower(originalPaymentMethod)

	// クレジット系の場合、マスターデータから最適なマッチを探す
	if strings.Contains(paymentLower, "クレジット") || strings.Contains(paymentLower, "credit") ||
		strings.Contains(paymentLower, "カード") || strings.Contains(paymentLower, "card") {

		// マスターデータから最適なカード系支払い方法を探す
		cardPayments := GetCardPaymentOptions(state)

		// 完全一致を優先
		for _, option := range cardPayments {
			optionLower := strings.ToLower(option.Label)
			if optionLower == paymentLower {
				log.Printf("支払い方法を詳細化: %s -> %s", originalPaymentMethod, option.Label)
				return option.Label
			}
		}

		// 部分一致を試す
		for _, option := range cardPayments {
			optionLower := strings.ToLower(option.Label)
			if strings.Contains(optionLower, paymentLower) || strings.Contains(paymentLower, optionLower) {
				log.Printf("支払い方法を詳細化（部分一致）: %s -> %s", originalPaymentMethod, option.Label)
				return option.Label
			}
		}

		// マッチしない場合は元の値を返す（後で手動選択可能）
		return originalPaymentMethod
	}

	// 電子マネー系の判定
	electronicMoney := []string{"suica", "pasmo", "icoca", "nanaco", "waon", "edy"}
	for _, keyword := range electronicMoney {
		if strings.Contains(paymentLower, keyword) {
			log.Printf("支払い方法を詳細化（電子マネー）: %s", originalPaymentMethod)
			return originalPaymentMethod
		}
	}

	// その他の場合は元の値を返す
	return originalPaymentMethod
}

// GetCardPaymentOptions はtype_kindがcardの支払い方法オプションを取得する
func GetCardPaymentOptions(state *models.BotState) []discordgo.SelectMenuOption {
	var options []discordgo.SelectMenuOption

	// type_kindからcardのIDを探す
	var cardTypeID int
	typeKindList := state.GetMasterData("type_kind").([]models.TypeKind)
	for _, typeKind := range typeKindList {
		if strings.ToLower(typeKind.TypeName) == "card" || typeKind.TypeName == "カード" {
			cardTypeID = typeKind.ID
			break
		}
	}

	// cardTypeIDに対応するtype_listのIDを探す
	var cardTypeListIDs []string
	typeList := state.GetMasterData("type_list").([]models.TypeList)
	for _, tl := range typeList {
		// type_listとtype_kindの関連を確認（IDが一致するかチェック）
		if typeKindId, err := strconv.Atoi(tl.ID); err == nil && typeKindId == cardTypeID {
			cardTypeListIDs = append(cardTypeListIDs, tl.ID)
		}
	}

	// cardTypeListIDsに対応するPaymentTypeを探す
	paymentTypes := state.GetMasterData("payment_types").([]models.PaymentType)
	for _, paymentType := range paymentTypes {
		for _, cardTypeListID := range cardTypeListIDs {
			if paymentType.TypeID == cardTypeListID {
				if len(options) < 25 { // Discord SelectMenuの制限
					options = append(options, discordgo.SelectMenuOption{
						Label: paymentType.PayKind,
						Value: strconv.Itoa(paymentType.PayID),
					})
				}
			}
		}
	}

	return options
}

// GetPaymentMethodByID はPaymentIDから支払い方法名を取得
func GetPaymentMethodByID(paymentID int, state *models.BotState) string {
	paymentTypes := state.GetMasterData("payment_types").([]models.PaymentType)
	for _, paymentType := range paymentTypes {
		if paymentType.PayID == paymentID {
			return paymentType.PayKind
		}
	}
	return "不明"
}

// FindPaymentIDByName は支払い方法名からPaymentIDを取得
func FindPaymentIDByName(paymentName string, state *models.BotState) *int {
	paymentTypes := state.GetMasterData("payment_types").([]models.PaymentType)
	for _, paymentType := range paymentTypes {
		if paymentType.PayKind == paymentName {
			return &paymentType.PayID
		}
	}
	return nil
}