package agent

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/generative-ai-go/genai"

	"github.com/yarikuri/errors"
	"github.com/yarikuri/models"
	"github.com/yarikuri/utils"
)

// AnalyzeReceiptInBackground は、バックグラウンドで画像解析を実行する
func AnalyzeReceiptInBackground(client *Client, m *discordgo.MessageCreate, state *models.TransactionState, tempImageDir string) {
	// 1. 画像をダウンロード
	imgPath, err := utils.DownloadImage(m.Attachments[0].URL, tempImageDir)
	if err != nil {
		log.Printf("画像ダウンロード失敗: %v", err)
		close(state.AIResultChan)
		return
	}
	state.ImagePath = imgPath

	// 2. AIに画像解析を依頼
	imgData, err := os.ReadFile(imgPath)
	if err != nil {
		log.Printf("画像読み込み失敗: %v", err)
		close(state.AIResultChan)
		return
	}

	prompt := GetReceiptAnalysisPrompt()
	ctx := context.Background()
	resp, err := client.GetModel().GenerateContent(ctx, genai.ImageData("png", imgData), prompt)
	if err != nil {
		botErr := errors.NewBotError(errors.ErrorTypeAIService, "Gemini APIレシート解析エラー", err).
			WithContext("user_id", m.Author.ID).
			WithContext("image_path", imgPath)
		errors.LogBotError(botErr)
		close(state.AIResultChan)
		return
	}

	// 3. 結果をパースしてチャネルに送信
	var analysisResult models.ReceiptAnalysis
	jsonStr := string(resp.Candidates[0].Content.Parts[0].(genai.Text))

	// JSONパース処理を実装
	log.Printf("Gemini API応答: %s", jsonStr)

	// 簡易的なパース（実際のレスポンス形式に応じて調整が必要）
	lines := strings.Split(jsonStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "日付:") {
			dateStr := strings.TrimSpace(strings.Split(line, ":")[1])
			if dateStr != "" && dateStr != "不明" {
				analysisResult.Date = &dateStr
			}
		}
		if strings.Contains(line, "金額:") {
			amountStr := strings.TrimSpace(strings.Split(line, ":")[1])
			if amount, err := strconv.Atoi(amountStr); err == nil {
				analysisResult.TotalAmount = &amount
			}
		}
		if strings.Contains(line, "支払い方法:") {
			paymentStr := strings.TrimSpace(strings.Split(line, ":")[1])
			if paymentStr != "" && paymentStr != "不明" {
				// クレジット系の場合、より詳細な分類を試みる
				enhancedPaymentMethod := EnhancePaymentMethod(paymentStr)
				analysisResult.PaymentMethod = &enhancedPaymentMethod
			}
		}
		if strings.Contains(line, "詳細:") {
			itemsStr := strings.TrimSpace(strings.Split(line, ":")[1])
			if itemsStr != "" && itemsStr != "不明" {
				analysisResult.Items = &itemsStr
			}
		}
	}

	// レシート判定：日付と金額が解析できた場合にtrue
	analysisResult.IsReceipt = (analysisResult.Date != nil && analysisResult.TotalAmount != nil)

	log.Printf("解析結果: IsReceipt=%t, Date=%v, Amount=%v",
		analysisResult.IsReceipt, analysisResult.Date, analysisResult.TotalAmount)

	state.AIResultChan <- analysisResult
}

// GenerateDetailWithAI はAIを使用して詳細説明を生成する
func GenerateDetailWithAI(client *Client, categoryName, storeName, items string) string {
	if client == nil {
		log.Printf("AIクライアントが初期化されていません")
		return generateFallbackDetail(storeName, items)
	}

	prompt := GetDetailGenerationPrompt(categoryName, storeName, items)
	ctx := context.Background()
	resp, err := client.GetModel().GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("詳細説明生成エラー: %v", err)
		// エラーの場合は従来の方式にフォールバック
		return generateFallbackDetail(storeName, items)
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		generatedText := string(resp.Candidates[0].Content.Parts[0].(genai.Text))
		// 生成されたテキストをクリーンアップ
		cleanedText := strings.TrimSpace(generatedText)
		cleanedText = strings.ReplaceAll(cleanedText, "\n", " ")
		cleanedText = strings.ReplaceAll(cleanedText, "\r", " ")

		if cleanedText != "" {
			log.Printf("LLMで詳細説明を生成: %s", cleanedText)
			return cleanedText
		}
	}

	// サンプルがない場合やLLM生成に失敗した場合はフォールバック
	return generateFallbackDetail(storeName, items)
}

// EnhancePaymentMethod は支払い方法の分類を強化する
func EnhancePaymentMethod(paymentStr string) string {
	paymentLower := strings.ToLower(paymentStr)

	// クレジットカード系の判定
	creditKeywords := []string{"visa", "master", "jcb", "amex", "american", "credit", "クレジット"}
	for _, keyword := range creditKeywords {
		if strings.Contains(paymentLower, keyword) {
			return "クレジットカード"
		}
	}

	// 電子マネー・QR決済系の判定
	electronicPayments := map[string]string{
		"paypay":    "PayPay",
		"rakuten":   "楽天ペイ",
		"楽天":        "楽天ペイ",
		"suica":     "Suica",
		"pasmo":     "PASMO",
		"icoca":     "ICOCA",
		"nanaco":    "nanaco",
		"waon":      "WAON",
		"edy":       "楽天Edy",
		"quicpay":   "QuicPay",
		"id":        "iD",
		"applepay":  "Apple Pay",
		"googlepay": "Google Pay",
		"linepay":   "LINE Pay",
		"aupay":     "au PAY",
		"dpay":      "d払い",
		"merpay":    "メルペイ",
	}

	for keyword, displayName := range electronicPayments {
		if strings.Contains(paymentLower, keyword) {
			return displayName
		}
	}

	// 現金判定
	cashKeywords := []string{"現金", "cash", "現金払い"}
	for _, keyword := range cashKeywords {
		if strings.Contains(paymentLower, keyword) {
			return "現金"
		}
	}

	// その他の場合は元の文字列をそのまま返す
	return paymentStr
}

// generateFallbackDetail はフォールバック用の詳細説明を生成
func generateFallbackDetail(storeName, items string) string {
	details := ""
	if items != "" {
		details = items
	}
	if storeName != "" {
		if details != "" {
			details += " - " + storeName
		} else {
			details = storeName
		}
	}
	if details == "" {
		details = "詳細不明"
	}
	return details
}