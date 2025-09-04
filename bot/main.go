// /home/ubuntu/Bot/discord/yarikuri/bot/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

// =================================================================================
// エラーハンドリング統一 
// =================================================================================

// BotError 統一エラー構造体
type BotError struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Cause   error                  `json:"cause,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ErrorType エラー分類
type ErrorType string

const (
	ErrorTypeDiscordAPI     ErrorType = "discord_api"
	ErrorTypeAIService      ErrorType = "ai_service"
	ErrorTypeDataAccess     ErrorType = "data_access"
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeConfiguration ErrorType = "configuration"
	ErrorTypeFileIO         ErrorType = "file_io"
	ErrorTypeNetwork        ErrorType = "network"
)

// NewBotError BotError生成関数
func NewBotError(errorType ErrorType, message string, cause error) *BotError {
	return &BotError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext コンテキスト情報追加
func (e *BotError) WithContext(key string, value interface{}) *BotError {
	e.Context[key] = value
	return e
}

// Error error インターフェース実装
func (e *BotError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// LogBotError 統一ログ出力関数
func LogBotError(err *BotError) {
	contextStr := ""
	if len(err.Context) > 0 {
		contextData, _ := json.Marshal(err.Context)
		contextStr = fmt.Sprintf(" context=%s", string(contextData))
	}
	
	log.Printf("BotError [%s] %s%s", err.Type, err.Message, contextStr)
	if err.Cause != nil {
		log.Printf("  -> Cause: %v", err.Cause)
	}
}

// HandleError 統一エラーハンドリング関数
func HandleError(err error, fallback func()) {
	if err == nil {
		return
	}
	
	if botErr, ok := err.(*BotError); ok {
		LogBotError(botErr)
	} else {
		log.Printf("Unexpected error: %v", err)
	}
	
	if fallback != nil {
		fallback()
	}
}

// =================================================================================
// グローバル変数定義
// =================================================================================
var (
	targetChannelID string
	geminiClient    *genai.GenerativeModel
	typeListMap     map[string]string
	typeKindMap     map[int]string
	transactions      map[string]*TransactionState // 進行中のトランザクションを管理
	confirmationData  map[string]*ConfirmationData // 確認画面のデータを管理
	mu                sync.Mutex                   // transactionsマップの同時アクセスを保護
	detailSamples     map[string]string            // カテゴリ名 -> 詳細説明サンプル

	masterCategories   []Category
	masterGroups       []Group
	masterPaymentTypes []PaymentType
	masterUsers        []User
	masterSourceList   []SourceList
	masterTypeKind     []TypeKind
	masterTypeList     []TypeList
	
	// マスターデータキュー管理
	masterDataQueues   map[string][]MasterQueueItem // マスターデータ種別ごとのキュー
	masterQueueMutex   sync.RWMutex                 // マスターデータキューの同期
)

const itemsPerPage = 15
const queueFilePath = "queue.json"
const tempImageDir = "./img"
const detailSamplesDir = "./detail_samples" // 詳細説明サンプルのディレクトリ

// =================================================================================
// 構造体定義
// =================================================================================
type Category struct { ID int; Name string }
type Group struct { ID int; Name string }
type PaymentType struct { PayID int; PayKind string; TypeID string }
type User struct { ID int; Name string }
type SourceList struct { ID int; SourceName string; TypeID int }
type TypeKind struct { ID int; TypeName string }
type TypeList struct { ID string; TypeName string }

type Expense struct {
	Date       string `json:"date"`
	Price      int    `json:"price"`
	CategoryID int    `json:"category_id"`
	UserID     int    `json:"user_id"`
	Detail     string `json:"detail"`
	GroupID    *int   `json:"group_id,omitempty"`
	PaymentID  *int   `json:"payment_id,omitempty"`
}

type ReceiptAnalysis struct {
	IsReceipt     bool    `json:"is_receipt"`
	StoreName     *string `json:"store_name"`
	Date          *string `json:"date"`
	TotalAmount   *int    `json:"total_amount"`
	PaymentMethod *string `json:"payment_method"`
	Items         *string `json:"items"`
}

type TransactionState struct {
	InitialMessageID string
	Interaction      *discordgo.InteractionCreate
	ImagePath        string
	UserInput        map[string]string
	AIResultChan     chan ReceiptAnalysis
}

type ConfirmationData struct {
	MessageID     string
	Date          string
	Amount        int
	CategoryID    int
	GroupID       *int
	UserID        int
	Detail        string
	PaymentMethod string
	AIResult      ReceiptAnalysis
}

// マスターデータキューアイテム
type MasterQueueItem struct {
	ID        string    `json:"id"`         // 一意識別子
	Type      string    `json:"type"`       // category, group, user, payment_type
	Name      string    `json:"name"`       // 追加するデータ名
	TypeName  string    `json:"type_name,omitempty"` // 支払い方法の場合のTypeName
	TypeID    string    `json:"type_id,omitempty"`   // 支払い方法の場合のTypeID (自動計算)
	Status    string    `json:"status"`     // pending, synced, error
	CreatedAt time.Time `json:"created_at"` // 作成日時
	UpdatedAt time.Time `json:"updated_at"` // 更新日時
}

// =================================================================================
// データ読み込み・解析関連
// =================================================================================
func loadMasterData(filePath string) error {
	log.Println("マスターデータのダンプファイルを読み込んでいます...")
	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		botErr := NewBotError(ErrorTypeFileIO, "マスターデータダンプファイルの読み込みに失敗", err).
			WithContext("file_path", filePath)
		return botErr
	}
	sqlContent := string(sqlBytes)
	
	records, _ := parseTableData(sqlContent, "category_list")
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0])); masterCategories = append(masterCategories, Category{ID: id, Name: strings.TrimSpace(rec[1])})
	}
	sort.Slice(masterCategories, func(i, j int) bool { return sortJapaneseFirst(masterCategories[i].Name, masterCategories[j].Name) })
	log.Printf("-> %d件のカテゴリを読み込み、ソートしました。\n", len(masterCategories))

	records, _ = parseTableData(sqlContent, "group_list")
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0])); masterGroups = append(masterGroups, Group{ID: id, Name: strings.TrimSpace(rec[1])})
	}
	sort.Slice(masterGroups, func(i, j int) bool { return sortJapaneseFirst(masterGroups[i].Name, masterGroups[j].Name) })
	log.Printf("-> %d件のグループを読み込み、ソートしました。\n", len(masterGroups))

	records, _ = parseTableData(sqlContent, "payment_type")
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0])); masterPaymentTypes = append(masterPaymentTypes, PaymentType{PayID: id, PayKind: strings.TrimSpace(rec[1]), TypeID: strings.TrimSpace(rec[2])})
	}
	sort.Slice(masterPaymentTypes, func(i, j int) bool { return sortJapaneseFirst(masterPaymentTypes[i].PayKind, masterPaymentTypes[j].PayKind) })
	log.Printf("-> %d件の支払い方法を読み込み、ソートしました。\n", len(masterPaymentTypes))
	
	records, _ = parseTableData(sqlContent, "user_list")
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0])); masterUsers = append(masterUsers, User{ID: id, Name: strings.TrimSpace(rec[1])})
	}
	sort.Slice(masterUsers, func(i, j int) bool { return sortJapaneseFirst(masterUsers[i].Name, masterUsers[j].Name) })
	log.Printf("-> %d件のユーザーを読み込み、ソートしました。\n", len(masterUsers))
	
	records, _ = parseTableData(sqlContent, "source_list")
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0])); typeId, _ := strconv.Atoi(strings.TrimSpace(rec[2])); masterSourceList = append(masterSourceList, SourceList{ID: id, SourceName: strings.TrimSpace(rec[1]), TypeID: typeId})
	}
	sort.Slice(masterSourceList, func(i, j int) bool { return sortJapaneseFirst(masterSourceList[i].SourceName, masterSourceList[j].SourceName) })
	log.Printf("-> %d件の収入源を読み込み、ソートしました。\n", len(masterSourceList))

	records, _ = parseTableData(sqlContent, "type_kind")
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0])); masterTypeKind = append(masterTypeKind, TypeKind{ID: id, TypeName: strings.TrimSpace(rec[1])})
	}
	typeKindMap = make(map[int]string)
	for _, item := range masterTypeKind { typeKindMap[item.ID] = item.TypeName }
	log.Printf("-> %d件の収入種別を読み込み、マップを作成しました。\n", len(masterTypeKind))

	records, _ = parseTableData(sqlContent, "type_list")
	for _, rec := range records {
		masterTypeList = append(masterTypeList, TypeList{ID: strings.TrimSpace(rec[0]), TypeName: strings.TrimSpace(rec[1])})
	}
	typeListMap = make(map[string]string)
	for _, item := range masterTypeList { typeListMap[item.ID] = item.TypeName }
	log.Printf("-> %d件の支払い種別を読み込み、マップを作成しました。\n", len(masterTypeList))

	log.Println("マスターデータの読み込みが完了しました。")
	return nil
}

// loadDetailSamples はマスターカテゴリーに基づいて詳細説明サンプルを読み込む
func loadDetailSamples(samplesDir string) error {
	log.Println("詳細説明サンプルを読み込んでいます...")
	detailSamples = make(map[string]string)
	
	// masterCategoriesが読み込まれていることを確認
	if len(masterCategories) == 0 {
		return NewBotError(ErrorTypeValidation, "マスターカテゴリーが読み込まれていません", nil).
			WithContext("required_action", "loadMasterData()を先に実行してください")
	}
	
	// 各カテゴリーに対応するtxtファイルを探す
	for _, category := range masterCategories {
		filePath := filepath.Join(samplesDir, category.Name+".txt")
		
		// ファイルが存在するかチェック
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("警告: カテゴリー「%s」のサンプルファイルが見つかりません: %s", category.Name, filePath)
			continue
		}
		
		// ファイルを読み込み
		content, err := os.ReadFile(filePath)
		if err != nil {
			botErr := NewBotError(ErrorTypeFileIO, "詳細サンプルファイルの読み込みに失敗", err).
				WithContext("file_path", filePath).
				WithContext("category", category.Name)
			LogBotError(botErr)
			continue
		}
		
		detailSamples[category.Name] = string(content)
		log.Printf("-> カテゴリー「%s」のサンプルを読み込みました", category.Name)
	}
	
	log.Printf("詳細説明サンプルの読み込みが完了しました。合計%d件", len(detailSamples))
	return nil
}

func parseTableData(sqlContent, tableName string) ([][]string, error) {
	startMarker := "COPY public." + tableName
	endMarker := "\\."
	startIndex := strings.Index(sqlContent, startMarker)
	if startIndex == -1 { return nil, nil }
	dataStartIndex := strings.Index(sqlContent[startIndex:], ";")
	if dataStartIndex == -1 { return nil, nil }
	dataBlockStartIndex := startIndex + dataStartIndex + 1
	endIndex := strings.Index(sqlContent[dataBlockStartIndex:], endMarker)
	if endIndex == -1 { return nil, nil }
	dataBlock := sqlContent[dataBlockStartIndex : dataBlockStartIndex+endIndex]
	lines := strings.Split(strings.TrimSpace(dataBlock), "\n")
	var records [][]string
	for _, line := range lines {
		if line != "" {
			records = append(records, strings.Split(line, "\t"))
		}
	}
	return records, nil
}

// =================================================================================
// Discordコマンド定義
// =================================================================================
var commands = []*discordgo.ApplicationCommand{
	{ Name: "check_master", Description: "メモリに読み込まれているマスターデータの件数を確認します。", },
	{
		Name: "show_master", Description: "指定したマスターデータのリストを表示します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString, Name: "type", Description: "表示したいマスターデータの種類", Required: true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "カテゴリ", Value: "category"}, {Name: "グループ", Value: "group"},
					{Name: "ユーザー", Value: "user"}, {Name: "支払い方法", Value: "payment_type"},
				},
			},
		},
	},
	{ Name: "add", Description: "レシートがない支出を手動で追加します。", },
	{
		Name: "fix", Description: "キューに追加された未同期のデータを修正します。",
		Options: []*discordgo.ApplicationCommandOption{
			{ Type: discordgo.ApplicationCommandOptionString, Name: "keyword", Description: "修正したいデータのキーワード", Required: true, },
		},
	},
	{
		Name: "add_master", Description: "新しいマスターデータを追加します。",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString, Name: "type", Description: "追加するマスターデータの種類", Required: true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "カテゴリ", Value: "category"}, {Name: "グループ", Value: "group"},
					{Name: "ユーザー", Value: "user"}, {Name: "支払い方法", Value: "payment_type"},
				},
			},
			{ Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "追加するデータの名前", Required: true, },
			{ Type: discordgo.ApplicationCommandOptionString, Name: "type_name", Description: "支払い方法の場合のみ：支払い種別（現金、クレジット等）", Required: false, },
		},
	},
}

var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"check_master": handleCheckMaster,
	"show_master":  handleShowMaster,
	"add":          handleAdd,
	"fix":          handleFix,
	"add_master":   handleAddMaster,
}

// =================================================================================
// Discordイベントハンドラ
// =================================================================================

// messageCreate は、画像投稿をトリガーに並行処理を開始する
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.ChannelID != targetChannelID || len(m.Attachments) == 0 {
		return
	}
	attachment := m.Attachments[0]
	if !strings.HasPrefix(attachment.ContentType, "image/") {
		return
	}

	log.Printf("画像を受信: %s", m.ID)

	// 1. 状態を初期化
	state := &TransactionState{
		InitialMessageID: m.ID,
		AIResultChan:     make(chan ReceiptAnalysis, 1),
	}
	mu.Lock()
	transactions[m.ID] = state
	mu.Unlock()

	// 2. バックグラウンドでAI解析を開始
	go analyzeReceiptInBackground(s, m, state)

	// 3. フォアグラウンドでユーザーに補足情報入力を求めるボタンを表示
	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content: "📋 レシートを解析中です...\n下のボタンをクリックして詳細情報を入力してください:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: "receipt_info_button:" + m.ID,
						Label:    "詳細情報を入力",
						Style:    discordgo.PrimaryButton,
						Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("補足情報ボタンの表示に失敗: %v", err)
	}
}

// (handleCheckMaster, handleShowMaster, handleFix, handlePagination は変更なし)
func handleCheckMaster(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title: "マスターデータ読み込み状況", Color: 0x00ff00, 
		Fields: []*discordgo.MessageEmbedField{
			{Name: "カテゴリ", Value: fmt.Sprintf("%d 件", len(masterCategories)), Inline: true},
			{Name: "グループ", Value: fmt.Sprintf("%d 件", len(masterGroups)), Inline: true},
			{Name: "ユーザー", Value: fmt.Sprintf("%d 件", len(masterUsers)), Inline: true},
			{Name: "支払い方法", Value: fmt.Sprintf("%d 件", len(masterPaymentTypes)), Inline: true},
			{Name: "収入源", Value: fmt.Sprintf("%d 件", len(masterSourceList)), Inline: true},
			{Name: "収入種別", Value: fmt.Sprintf("%d 件", len(masterTypeKind)), Inline: true},
			{Name: "支払い種別", Value: fmt.Sprintf("%d 件", len(masterTypeList)), Inline: true},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}
func handleShowMaster(s *discordgo.Session, i *discordgo.InteractionCreate) {
	dataType := i.ApplicationCommandData().Options[0].StringValue()
	embed, components, err := generatePaginatedData(dataType, 0)
	if err != nil {
		botErr := NewBotError(ErrorTypeDataAccess, "ページデータ生成エラー", err).
			WithContext("data_type", dataType).
			WithContext("page", 0)
		LogBotError(botErr)
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{ Embeds: []*discordgo.MessageEmbed{embed}, Components: components, },
	})
}
func handleFix(s *discordgo.Session, i *discordgo.InteractionCreate) {
	keyword := i.ApplicationCommandData().Options[0].StringValue()
	responseText := fmt.Sprintf("「%s」でキュー内のデータを検索します... (現在は検索機能は未実装です)", keyword)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseText, Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleReceiptInfoButton はボタンクリック時にカテゴリー選択画面を表示する
func handleReceiptInfoButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "receipt_info_button:")
	
	// カテゴリー選択用のSelectMenuオプションを準備（最大25件）
	var categoryOptions []discordgo.SelectMenuOption
	for _, category := range masterCategories {
		if len(categoryOptions) >= 25 {
			break // Discord SelectMenuの制限
		}
		categoryOptions = append(categoryOptions, discordgo.SelectMenuOption{
			Label: category.Name,
			Value: strconv.Itoa(category.ID),
		})
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "📋 まずカテゴリーを選択してください:",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "category_select:" + messageID,
							Placeholder: "カテゴリーを選択...",
							Options:     categoryOptions,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							CustomID: "category_search:" + messageID,
							Label:    "🔍 キーワード検索",
							Style:    discordgo.SecondaryButton,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("カテゴリー選択画面の表示エラー: %v", err)
	}
}

// handleCategorySelect はカテゴリー選択後にモーダルを表示する
func handleCategorySelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "category_select:")
	
	selectedCategoryID := i.MessageComponentData().Values[0]
	
	// 選択されたカテゴリー名を取得（将来使用予定）
	// var selectedCategoryName string
	// for _, category := range masterCategories {
	// 	if strconv.Itoa(category.ID) == selectedCategoryID {
	// 		selectedCategoryName = category.Name
	// 		break
	// 	}
	// }

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "receipt_info_modal:" + messageID,
			Title:    "レシート情報の補足",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "price",
						Label:       "金額 (総額と違う場合のみ入力)",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "例: 1200",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "category_id",
						Label:       "選択されたカテゴリー",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Value:       selectedCategoryID,
						MaxLength:   10,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "group_keyword",
						Label:       "グループ検索キーワード (任意)",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "例: 外食",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "user_name",
						Label:       "支払者名 (空白で自分)",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "例: 自分, 田中, 木星",
						Value:       "自分",
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("モーダル表示エラー: %v", err)
	}
}

// handleCategorySearch はキーワード検索モーダルを表示する
func handleCategorySearch(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "category_search:")

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "category_search_modal:" + messageID,
			Title:    "カテゴリー検索",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "search_keyword",
						Label:       "検索キーワードを入力",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: "例: 食, ごはん, 交通",
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("検索モーダル表示エラー: %v", err)
	}
}

// handleCategorySearchModal はキーワード検索結果を表示する
func handleCategorySearchModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	messageID := strings.TrimPrefix(customID, "category_search_modal:")
	
	// 検索キーワードを取得
	var searchKeyword string
	for _, row := range i.ModalSubmitData().Components {
		for _, component := range row.(*discordgo.ActionsRow).Components {
			textInput := component.(*discordgo.TextInput)
			if textInput.CustomID == "search_keyword" {
				searchKeyword = textInput.Value
				break
			}
		}
	}
	
	// カテゴリーを検索
	matchedCategories := searchCategories(searchKeyword)
	
	if len(matchedCategories) == 0 {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "該当するカテゴリーが見つかりませんでした。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("検索結果応答エラー: %v", err)
		}
		return
	}

	// 検索結果をSelectMenuで表示（最大25件）
	var categoryOptions []discordgo.SelectMenuOption
	for i, category := range matchedCategories {
		if i >= 25 {
			break
		}
		categoryOptions = append(categoryOptions, discordgo.SelectMenuOption{
			Label: category.Name,
			Value: strconv.Itoa(category.ID),
		})
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🔍 「%s」の検索結果 (%d件):", searchKeyword, len(matchedCategories)),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "category_select:" + messageID,
							Placeholder: "カテゴリーを選択...",
							Options:     categoryOptions,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("検索結果表示エラー: %v", err)
	}
}

// searchCategories はキーワードに基づいてカテゴリーを検索する
func searchCategories(keyword string) []Category {
	keyword = strings.ToLower(keyword)
	var matched []Category
	
	for _, category := range masterCategories {
		categoryName := strings.ToLower(category.Name)
		
		// 完全一致を優先
		if categoryName == keyword {
			matched = append([]Category{category}, matched...)
			continue
		}
		
		// 部分一致
		if strings.Contains(categoryName, keyword) || strings.Contains(keyword, categoryName) {
			matched = append(matched, category)
			continue
		}
		
		// キーワードマッチング（食事系など）
		if matchCategoryKeywords(categoryName, keyword) {
			matched = append(matched, category)
		}
	}
	
	return matched
}

// matchCategoryKeywords はカテゴリーとキーワードの関連性をチェック
func matchCategoryKeywords(categoryName, keyword string) bool {
	foodKeywords := []string{"食", "飯", "料理", "ごはん", "めし", "たべもの"}
	transportKeywords := []string{"交通", "電車", "バス", "タクシー", "移動"}
	
	categoryContainsFood := false
	categoryContainsTransport := false
	
	for _, foodKW := range foodKeywords {
		if strings.Contains(categoryName, foodKW) {
			categoryContainsFood = true
			break
		}
	}
	
	for _, transportKW := range transportKeywords {
		if strings.Contains(categoryName, transportKW) {
			categoryContainsTransport = true
			break
		}
	}
	
	keywordIsFood := false
	keywordIsTransport := false
	
	for _, foodKW := range foodKeywords {
		if strings.Contains(keyword, foodKW) {
			keywordIsFood = true
			break
		}
	}
	
	for _, transportKW := range transportKeywords {
		if strings.Contains(keyword, transportKW) {
			keywordIsTransport = true
			break
		}
	}
	
	return (categoryContainsFood && keywordIsFood) || (categoryContainsTransport && keywordIsTransport)
}

// handleReceiptInfoModal はモーダル送信を処理する
func handleReceiptInfoModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	messageID := strings.TrimPrefix(customID, "receipt_info_modal:")
	
	// モーダルデータを取得
	modalData := i.ModalSubmitData().Components
	userInput := make(map[string]string)
	
	for _, row := range modalData {
		for _, component := range row.(*discordgo.ActionsRow).Components {
			textInput := component.(*discordgo.TextInput)
			userInput[textInput.CustomID] = textInput.Value
		}
	}
	
	log.Printf("モーダルデータを受信: messageID=%s, data=%+v", messageID, userInput)
	
	// 一時的に応答を送信
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "📋 入力情報を受信しました。AI解析結果と合わせて処理中...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("応答エラー: %v", err)
	}
	
	// バックグラウンド処理と結合してデータベースに保存する処理をここに追加
	go processReceiptWithUserInput(s, messageID, userInput)
}

// processReceiptWithUserInput はユーザー入力とAI解析結果を組み合わせて処理する
func processReceiptWithUserInput(s *discordgo.Session, messageID string, userInput map[string]string) {
	// トランザクション状態を取得
	mu.Lock()
	state, exists := transactions[messageID]
	mu.Unlock()
	
	if !exists {
		log.Printf("トランザクション状態が見つかりません: %s", messageID)
		return
	}
	
	// AI解析結果を待機（タイムアウト付き）
	select {
	case aiResult := <-state.AIResultChan:
		log.Printf("AI解析結果とユーザー入力を結合中: messageID=%s", messageID)
		
		// カテゴリーをIDから決定（新しい選択方式）
		var categoryID int
		if categoryIDStr := userInput["category_id"]; categoryIDStr != "" {
			// SelectMenuから選択されたカテゴリーID
			if cid, err := strconv.Atoi(categoryIDStr); err == nil {
				categoryID = cid
			} else {
				categoryID = 1 // デフォルト値
			}
		} else {
			// 旧方式のキーワード検索（フォールバック）
			categoryKeyword := userInput["category_keyword"]
			categoryID = findCategoryByKeyword(categoryKeyword)
		}
		
		// グループをキーワードから決定（任意）
		var groupID *int
		if groupKeyword := userInput["group_keyword"]; groupKeyword != "" {
			if gid := findGroupByKeyword(groupKeyword); gid != nil {
				groupID = gid
			}
		}
		
		// ユーザー処理（名前ベース、デフォルトは「自分」でID=0）
		var userID int = 0 // デフォルトは「自分」のID=0
		if userName := userInput["user_name"]; userName != "" && userName != "自分" {
			// ユーザー名で検索
			for _, user := range masterUsers {
				if strings.Contains(user.Name, userName) || strings.Contains(userName, user.Name) {
					userID = user.ID
					break
				}
			}
		}
		
		// 金額処理（ユーザー入力があれば優先、なければAI解析結果を使用）
		var amount int
		if aiResult.TotalAmount != nil {
			amount = *aiResult.TotalAmount
		}
		if priceStr := userInput["price"]; priceStr != "" {
			if userAmount, err := strconv.Atoi(priceStr); err == nil {
				amount = userAmount
			}
		}
		
		// 詳細説明を生成
		detail := generateDetailFromSamples(categoryID, aiResult)
		
		log.Printf("処理結果 - Amount: %d, Category: %d, Group: %v, User: %d, Detail: %s",
			amount, categoryID, groupID, userID, detail)
		
		// 処理完了をチャンネルに通知
		go sendProcessingResult(s, state.InitialMessageID, amount, categoryID, groupID, userID, detail, aiResult)
		
	case <-time.After(30 * time.Second):
		log.Printf("AI解析がタイムアウトしました: %s", messageID)
	}
	
	// 状態をクリーンアップ
	mu.Lock()
	delete(transactions, messageID)
	mu.Unlock()
}

// findCategoryByKeyword はキーワードからカテゴリーIDを見つける
func findCategoryByKeyword(keyword string) int {
	keyword = strings.ToLower(keyword)
	log.Printf("カテゴリー検索: %s", keyword)
	
	for _, category := range masterCategories {
		categoryName := strings.ToLower(category.Name)
		log.Printf("  比較中: %s", categoryName)
		
		// 完全一致
		if categoryName == keyword {
			log.Printf("  → 完全一致: %s (ID: %d)", category.Name, category.ID)
			return category.ID
		}
		
		// 部分一致（双方向）
		if strings.Contains(categoryName, keyword) || strings.Contains(keyword, categoryName) {
			log.Printf("  → 部分一致: %s (ID: %d)", category.Name, category.ID)
			return category.ID
		}
		
		// キーワードによる推測マッチング
		if (strings.Contains(keyword, "食") || strings.Contains(keyword, "飯") || strings.Contains(keyword, "料理")) &&
		   (strings.Contains(categoryName, "食") || strings.Contains(categoryName, "飯") || strings.Contains(categoryName, "料理")) {
			log.Printf("  → 食事系マッチング: %s (ID: %d)", category.Name, category.ID)
			return category.ID
		}
	}
	
	log.Printf("  → デフォルトカテゴリーを使用: ID=1")
	return 1 // デフォルトカテゴリー
}

// findGroupByKeyword はキーワードからグループIDを見つける
func findGroupByKeyword(keyword string) *int {
	keyword = strings.ToLower(keyword)
	log.Printf("グループ検索: %s", keyword)
	
	for _, group := range masterGroups {
		groupName := strings.ToLower(group.Name)
		log.Printf("  比較中: %s", groupName)
		
		// 完全一致
		if groupName == keyword {
			log.Printf("  → 完全一致: %s (ID: %d)", group.Name, group.ID)
			return &group.ID
		}
		
		// 部分一致（双方向）
		if strings.Contains(groupName, keyword) || strings.Contains(keyword, groupName) {
			log.Printf("  → 部分一致: %s (ID: %d)", group.Name, group.ID)
			return &group.ID
		}
	}
	
	log.Printf("  → グループが見つかりません")
	return nil
}

// generateDetailFromSamples はLLMを使用してカテゴリー別の詳細説明を生成する
func generateDetailFromSamples(categoryID int, aiResult ReceiptAnalysis) string {
	// カテゴリー名を取得
	var categoryName string
	for _, category := range masterCategories {
		if category.ID == categoryID {
			categoryName = category.Name
			break
		}
	}
	
	// detail_samplesから該当カテゴリーのサンプルを探す
	samplePattern, hasSample := detailSamples[categoryName]
	
	// AI解析結果から基本情報を抽出
	var storeName, items, paymentMethod string
	if aiResult.StoreName != nil {
		storeName = *aiResult.StoreName
	}
	if aiResult.Items != nil {
		items = *aiResult.Items
	}
	if aiResult.PaymentMethod != nil {
		paymentMethod = *aiResult.PaymentMethod
	}
	
	// サンプルパターンがある場合はLLMで詳細説明を生成
	if hasSample {
		prompt := fmt.Sprintf(`あなたは家計簿の詳細説明を生成するアシスタントです。

以下の情報に基づいて、「%s」カテゴリーの詳細説明を生成してください。

【レシート情報】
店舗名: %s
商品/サービス: %s
支払い方法: %s

【このカテゴリーの入力パターンサンプル】
%s

【生成ルール】
1. サンプルパターンに従った形式で記述してください
2. 店舗名と商品名は正確に記載してください
3. 簡潔で分かりやすい表現にしてください
4. 日本語で記述してください
5. 特殊記号や改行は使用せず、一行で記述してください

詳細説明:`, categoryName, storeName, items, paymentMethod, samplePattern)

		// Gemini APIで詳細説明を生成
		ctx := context.Background()
		resp, err := geminiClient.GenerateContent(ctx, genai.Text(prompt))
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
	}
	
	// サンプルがない場合やLLM生成に失敗した場合はフォールバック
	return generateFallbackDetail(storeName, items)
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
		details = "レシート解析結果"
	}
	return details
}

// sendProcessingResult はキュー追加前の確認画面を表示する
func sendProcessingResult(s *discordgo.Session, messageID string, amount int, categoryID int, groupID *int, userID int, detail string, aiResult ReceiptAnalysis) {
	// カテゴリー名を取得
	var categoryName string = "不明"
	for _, category := range masterCategories {
		if category.ID == categoryID {
			categoryName = category.Name
			break
		}
	}
	
	// グループ名を取得
	var groupName string = "なし"
	if groupID != nil {
		for _, group := range masterGroups {
			if group.ID == *groupID {
				groupName = group.Name
				break
			}
		}
	}
	
	// ユーザー名を取得
	var userName string = "不明"
	for _, user := range masterUsers {
		if user.ID == userID {
			userName = user.Name
			break
		}
	}
	
	// 日付情報
	var dateStr string = "不明"
	if aiResult.Date != nil {
		dateStr = *aiResult.Date
	} else {
		dateStr = time.Now().Format("2006-01-02")
	}
	
	// 支払い方法情報を取得
	var paymentMethod string = "不明"
	if aiResult.PaymentMethod != nil {
		paymentMethod = *aiResult.PaymentMethod
	}
	
	// データを一時保存用の構造体に格納
	storeConfirmationData(messageID, amount, categoryID, groupID, userID, detail, dateStr, paymentMethod, aiResult)
	
	// Embedを作成（確認画面用）
	embed := &discordgo.MessageEmbed{
		Title: "📋 キューに追加前の確認",
		Color: 0xffa500,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📅 日付", Value: dateStr, Inline: true},
			{Name: "💵 金額", Value: fmt.Sprintf("¥%d", amount), Inline: true},
			{Name: "💳 支払い方法", Value: paymentMethod, Inline: true},
			{Name: "📂 カテゴリー", Value: categoryName, Inline: true},
			{Name: "🏷️ グループ", Value: groupName, Inline: true},
			{Name: "👤 支払者", Value: userName, Inline: true},
			{Name: "📝 詳細", Value: detail, Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "各項目を編集できます。問題なければ「キューに追加」をクリックしてください。",
		},
	}
	
	// 編集ボタンを作成
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_date:%s", messageID),
					Label:    "📅 日付を編集",
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_amount:%s", messageID),
					Label:    "💵 金額を編集",
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_payment:%s", messageID),
					Label:    "💳 支払い方法を編集",
					Style:    discordgo.SecondaryButton,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_group:%s", messageID),
					Label:    "🏷️ グループを編集",
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_payer:%s", messageID),
					Label:    "👤 支払者を編集",
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_detail:%s", messageID),
					Label:    "📝 詳細を編集",
					Style:    discordgo.SecondaryButton,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("add_to_queue:%s", messageID),
					Label:    "✅ キューに追加",
					Style:    discordgo.SuccessButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("cancel_entry:%s", messageID),
					Label:    "❌ キャンセル",
					Style:    discordgo.DangerButton,
				},
			},
		},
	}
	
	// メッセージを送信
	_, err := s.ChannelMessageSendComplex(targetChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Printf("確認画面の送信に失敗: %v", err)
	} else {
		log.Printf("確認画面を送信しました: messageID=%s", messageID)
	}
}

// storeConfirmationData は確認画面のデータを一時保存する
func storeConfirmationData(messageID string, amount int, categoryID int, groupID *int, userID int, detail, date, paymentMethod string, aiResult ReceiptAnalysis) {
	mu.Lock()
	defer mu.Unlock()
	
	if confirmationData == nil {
		confirmationData = make(map[string]*ConfirmationData)
	}
	
	confirmationData[messageID] = &ConfirmationData{
		MessageID:     messageID,
		Date:          date,
		Amount:        amount,
		CategoryID:    categoryID,
		GroupID:       groupID,
		UserID:        userID,
		Detail:        detail,
		PaymentMethod: paymentMethod,
		AIResult:      aiResult,
	}
}

// getConfirmationData は確認画面のデータを取得する
func getConfirmationData(messageID string) *ConfirmationData {
	mu.Lock()
	defer mu.Unlock()
	
	if confirmationData == nil {
		return nil
	}
	
	return confirmationData[messageID]
}

// updateConfirmationData は確認画面のデータを更新する
func updateConfirmationData(messageID string, updateFunc func(*ConfirmationData)) {
	mu.Lock()
	defer mu.Unlock()
	
	if confirmationData == nil {
		return
	}
	
	if data, exists := confirmationData[messageID]; exists {
		updateFunc(data)
	}
	
}

func handlePagination(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{ Type: discordgo.InteractionResponseDeferredMessageUpdate, })
	if err != nil { log.Printf("遅延応答エラー: %v", err); return }
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) != 3 { log.Printf("CustomID形式エラー: %s", customID); return }
	dataType := parts[1]
	page, err := strconv.Atoi(parts[2])
	if err != nil { log.Printf("ページ番号解析エラー: %v", err); return }
	embed, components, err := generatePaginatedData(dataType, page)
	if err != nil { log.Printf("ページデータ生成エラー: %v", err); return }
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{ Embeds: &[]*discordgo.MessageEmbed{embed}, Components: &components, })
	if err != nil { log.Printf("メッセージ更新エラー: %v", err) }
}

// handleAdd は /add コマンドの処理
func handleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var typeOptions []discordgo.SelectMenuOption
	for _, typeItem := range masterTypeList {
		typeOptions = append(typeOptions, discordgo.SelectMenuOption{ Label: typeItem.TypeName, Value: typeItem.ID, })
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "add_modal_step1", Title: "手動データ追加 (ステップ1/2)",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{ Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "date", Label: "日付 (YYYY-MM-DD)", Style: discordgo.TextInputShort,
						Placeholder: "例: " + time.Now().Format("2006-01-02"), Required: true, Value: time.Now().Format("2006-01-02"),
					},
				}},
				discordgo.ActionsRow{ Components: []discordgo.MessageComponent{
					discordgo.TextInput{ CustomID: "price", Label: "金額", Style: discordgo.TextInputShort, Placeholder: "例: 1280", Required: true, },
				}},
				discordgo.ActionsRow{ Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "category_keyword", Label: "カテゴリ検索キーワード", Style: discordgo.TextInputShort,
						Placeholder: "例: ごはん, 交通", Required: true,
					},
				}},
				discordgo.ActionsRow{ Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "group_keyword", Label: "グループ検索キーワード (任意)", Style: discordgo.TextInputShort,
						Placeholder: "例: 東北旅行", Required: false,
					},
				}},
				discordgo.ActionsRow{ Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{ CustomID: "payment_type_select", Placeholder: "支払い種別を選択", Options: typeOptions, },
				}},
			},
		},
	})
	if err != nil { log.Printf("モーダル表示エラー: %v", err) }
}

// =================================================================================
// 編集モーダル送信ハンドラー
// =================================================================================

// handleEditDateModal は日付編集モーダルの送信を処理する
func handleEditDateModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_date_modal:")
	
	// 入力値を取得
	var newDate string
	for _, row := range i.ModalSubmitData().Components {
		for _, component := range row.(*discordgo.ActionsRow).Components {
			textInput := component.(*discordgo.TextInput)
			if textInput.CustomID == "date" {
				newDate = textInput.Value
				break
			}
		}
	}
	
	// 日付フォーマットを検証
	_, err := time.Parse("2006-01-02", newDate)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 日付の形式が正しくありません。YYYY-MM-DD形式で入力してください。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	// データを更新
	updateConfirmationData(messageID, func(data *ConfirmationData) {
		data.Date = newDate
	})
	
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ 日付を %s に更新しました。", newDate),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	
	// 確認画面を更新
	updateConfirmationDisplay(s, messageID)
}

// handleEditAmountModal は金額編集モーダルの送信を処理する
func handleEditAmountModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_amount_modal:")
	
	// 入力値を取得
	var amountStr string
	for _, row := range i.ModalSubmitData().Components {
		for _, component := range row.(*discordgo.ActionsRow).Components {
			textInput := component.(*discordgo.TextInput)
			if textInput.CustomID == "amount" {
				amountStr = textInput.Value
				break
			}
		}
	}
	
	// 金額を数値に変換
	newAmount, err := strconv.Atoi(amountStr)
	if err != nil || newAmount < 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ 金額は正の整数で入力してください。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	// データを更新
	updateConfirmationData(messageID, func(data *ConfirmationData) {
		data.Amount = newAmount
	})
	
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ 金額を ¥%d に更新しました。", newAmount),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	
	// 確認画面を更新
	updateConfirmationDisplay(s, messageID)
}

// handleEditPaymentModal は支払い方法編集モーダルの送信を処理する
func handleEditPaymentModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_payment_modal:")
	
	// 入力値を取得
	var newPaymentMethod string
	for _, row := range i.ModalSubmitData().Components {
		for _, component := range row.(*discordgo.ActionsRow).Components {
			textInput := component.(*discordgo.TextInput)
			if textInput.CustomID == "payment_method" {
				newPaymentMethod = textInput.Value
				break
			}
		}
	}
	
	// データを更新
	updateConfirmationData(messageID, func(data *ConfirmationData) {
		data.PaymentMethod = newPaymentMethod
	})
	
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ 支払い方法を「%s」に更新しました。", newPaymentMethod),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	
	// 確認画面を更新
	updateConfirmationDisplay(s, messageID)
}

// handleEditDetailModal は詳細編集モーダルの送信を処理する
func handleEditDetailModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_detail_modal:")
	
	// 入力値を取得
	var newDetail string
	for _, row := range i.ModalSubmitData().Components {
		for _, component := range row.(*discordgo.ActionsRow).Components {
			textInput := component.(*discordgo.TextInput)
			if textInput.CustomID == "detail" {
				newDetail = textInput.Value
				break
			}
		}
	}
	
	// データを更新
	updateConfirmationData(messageID, func(data *ConfirmationData) {
		data.Detail = newDetail
	})
	
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ 詳細を更新しました。",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	
	// 確認画面を更新
	updateConfirmationDisplay(s, messageID)
}

// =================================================================================
// 確認画面編集ハンドラー
// =================================================================================

// handleEditDate は日付編集モーダルを表示する
func handleEditDate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_date:")
	
	data := getConfirmationData(messageID)
	if data == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: データが見つかりません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "edit_date_modal:" + messageID,
			Title:    "日付を編集",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "date",
						Label:       "日付 (YYYY-MM-DD形式)",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Value:       data.Date,
						Placeholder: "例: 2025-08-24",
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("日付編集モーダル表示エラー: %v", err)
	}
}

// handleEditAmount は金額編集モーダルを表示する
func handleEditAmount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_amount:")
	
	data := getConfirmationData(messageID)
	if data == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: データが見つかりません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "edit_amount_modal:" + messageID,
			Title:    "金額を編集",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "amount",
						Label:       "金額（数字のみ）",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Value:       strconv.Itoa(data.Amount),
						Placeholder: "例: 1500",
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("金額編集モーダル表示エラー: %v", err)
	}
}

// handleEditPayment は支払い方法編集モーダルを表示する
func handleEditPayment(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_payment:")
	
	data := getConfirmationData(messageID)
	if data == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: データが見つかりません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "edit_payment_modal:" + messageID,
			Title:    "支払い方法を編集",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "payment_method",
						Label:       "支払い方法",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Value:       data.PaymentMethod,
						Placeholder: "例: クレジット, 現金, デビット",
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("支払い方法編集モーダル表示エラー: %v", err)
	}
}

// handleEditGroup はグループ編集用のセレクトメニューを表示する
func handleEditGroup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_group:")
	
	data := getConfirmationData(messageID)
	if data == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: データが見つかりません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	// グループ選択用のSelectMenuオプションを準備（最大25件）
	var groupOptions []discordgo.SelectMenuOption
	groupOptions = append(groupOptions, discordgo.SelectMenuOption{
		Label: "なし",
		Value: "none",
	})
	
	for _, group := range masterGroups {
		if len(groupOptions) >= 25 {
			break
		}
		groupOptions = append(groupOptions, discordgo.SelectMenuOption{
			Label: group.Name,
			Value: strconv.Itoa(group.ID),
		})
	}
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🏷️ グループを選択してください:",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "group_select:" + messageID,
							Placeholder: "グループを選択...",
							Options:     groupOptions,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("グループ編集メニュー表示エラー: %v", err)
	}
}

// handleEditPayer は支払者編集用のセレクトメニューを表示する
func handleEditPayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_payer:")
	
	data := getConfirmationData(messageID)
	if data == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: データが見つかりません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	// ユーザー選択用のSelectMenuオプションを準備（最大25件）
	var userOptions []discordgo.SelectMenuOption
	for _, user := range masterUsers {
		if len(userOptions) >= 25 {
			break
		}
		userOptions = append(userOptions, discordgo.SelectMenuOption{
			Label: user.Name,
			Value: strconv.Itoa(user.ID),
		})
	}
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "👤 支払者を選択してください:",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "payer_select:" + messageID,
							Placeholder: "支払者を選択...",
							Options:     userOptions,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("支払者編集メニュー表示エラー: %v", err)
	}
}

// handleEditDetail は詳細編集モーダルを表示する
func handleEditDetail(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "edit_detail:")
	
	data := getConfirmationData(messageID)
	if data == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "エラー: データが見つかりません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "edit_detail_modal:" + messageID,
			Title:    "詳細を編集",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "detail",
						Label:       "詳細",
						Style:       discordgo.TextInputParagraph,
						Required:    true,
						Value:       data.Detail,
						Placeholder: "店舗名や購入商品の詳細を入力...",
						MaxLength:   500,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("詳細編集モーダル表示エラー: %v", err)
	}
}

// =================================================================================
// ヘルパー関数
// =================================================================================
func generatePaginatedData(dataType string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	var allItems []string
	var title string
	switch dataType {
	case "category":
		title = "カテゴリ一覧"
		categoriesWithQueue := getMasterDataWithQueue("category").([]Category)
		for _, item := range categoriesWithQueue { allItems = append(allItems, item.Name) }
	case "group":
		title = "グループ一覧"
		groupsWithQueue := getMasterDataWithQueue("group").([]Group)
		for _, item := range groupsWithQueue { allItems = append(allItems, item.Name) }
	case "user":
		title = "ユーザー一覧"
		usersWithQueue := getMasterDataWithQueue("user").([]User)
		for _, item := range usersWithQueue { allItems = append(allItems, item.Name) }
	case "payment_type":
		title = "支払い方法一覧"
		paymentsWithQueue := getMasterDataWithQueue("payment_type").([]PaymentType)
		for _, item := range paymentsWithQueue {
			typeName := typeListMap[item.TypeID];
			if typeName == "" { typeName = "不明" };
			allItems = append(allItems, fmt.Sprintf("%s (%s)", item.PayKind, typeName))
		}
	case "source_list":
		title = "収入源一覧"
		for _, item := range masterSourceList {
			typeName := typeKindMap[item.TypeID];
			if typeName == "" { typeName = "不明" };
			allItems = append(allItems, fmt.Sprintf("%s (%s)", item.SourceName, typeName))
		}
	default:
		return nil, nil, fmt.Errorf("不明なデータタイプです: %s", dataType)
	}

	start, end := calculatePageBounds(page, len(allItems))
	pageItems := allItems[start:end]
	totalPages := (len(allItems) + itemsPerPage - 1) / itemsPerPage

	embed := &discordgo.MessageEmbed{
		Title: title, Description: strings.Join(pageItems, "\n"), Color: 0x00aaff,
		Footer: &discordgo.MessageEmbedFooter{ Text: fmt.Sprintf("ページ %d / %d", page+1, totalPages), },
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "◀", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("paginate:%s:%d", dataType, page-1), Disabled: page == 0,
				},
				discordgo.Button{
					Label: "▶", Style: discordgo.PrimaryButton, CustomID: fmt.Sprintf("paginate:%s:%d", dataType, page+1), Disabled: page+1 >= totalPages,
				},
			},
		},
	}
	return embed, components, nil
}
func calculatePageBounds(page, totalItems int) (int, int) {
	start := page * itemsPerPage
	end := start + itemsPerPage
	if start >= totalItems { start = totalItems }
	if end > totalItems { end = totalItems }
	return start, end
}
func isJapanese(s string) bool {
	for _, r := range s { if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) { return true } }; return false
}
func sortJapaneseFirst(s1, s2 string) bool {
	isJp1, isJp2 := isJapanese(s1), isJapanese(s2)
	if isJp1 != isJp2 { return isJp1 }; return s1 < s2
}
func downloadImage(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", NewBotError(ErrorTypeNetwork, "画像URLへのHTTPリクエストに失敗", err).
			WithContext("url", url)
	}
	defer response.Body.Close()

	os.MkdirAll(tempImageDir, 0755)

	filePath := filepath.Join(tempImageDir, filepath.Base(response.Request.URL.Path))
	file, err := os.Create(filePath)
	if err != nil {
		return "", NewBotError(ErrorTypeFileIO, "一時画像ファイルの作成に失敗", err).
			WithContext("file_path", filePath)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", NewBotError(ErrorTypeFileIO, "画像データの書き込みに失敗", err).
			WithContext("file_path", filePath)
	}

	return filePath, nil
}

// analyzeReceiptInBackground は、バックグラウンドで画像解析を実行する
func analyzeReceiptInBackground(s *discordgo.Session, m *discordgo.MessageCreate, state *TransactionState) {
	// 1. 画像をダウンロード
	imgPath, err := downloadImage(m.Attachments[0].URL)
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

	prompt := genai.Text(`あなたはレシート情報抽出アシスタントです。
添付されたレシート画像から、以下の情報を指定されたフォーマットで正確に書き出してください。

日付: [yyyy/mm/dd形式または省略形式]
金額: [金額（整数または小数）]
支払い方法: [クレジット/現金/その他の支払い方法]
カテゴリー: [御飯代/交通費/その他のカテゴリー]
グループ: [グループ名またはnull]
ユーザー: [ユーザー名]
詳細: [店舗名や購入商品の詳細情報]

**重要ルール：**
1. 日付はyyyy/mm/dd形式で記載してください。ただし、月や日が一桁の場合は0を省略してもよい（例: 2025/8/19 や 2025-8-19）
2. 金額は整数または小数で記載してください（円マークは不要）
3. 支払い方法でカードやクレジット系の場合は「クレジット」と記載してください
4. グループに該当する情報がない場合は「null」と記載してください
5. ユーザーは基本的に「自分」としてください
6. 詳細には店舗名や購入した商品名を含めてください
7. 見えない・読み取れない部分は「不明」と記載してください`)
	
	ctx := context.Background()
	resp, err := geminiClient.GenerateContent(ctx, genai.ImageData("png", imgData), prompt)
	if err != nil {
		botErr := NewBotError(ErrorTypeAIService, "Gemini APIレシート解析エラー", err).
			WithContext("user_id", m.Author.ID).
			WithContext("image_path", imgPath)
		LogBotError(botErr)
		close(state.AIResultChan)
		return
	}

	// 3. 結果をパースしてチャネルに送信
	var analysisResult ReceiptAnalysis
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
				analysisResult.PaymentMethod = &paymentStr
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

// =================================================================================
// main関数 (Botの起動)
// =================================================================================
func main() {
	err := godotenv.Load()
	if err != nil { log.Println("Note: .env file not found, continuing without it.") }

	targetChannelID = os.Getenv("CHANNEL_ID")
	if targetChannelID == "" {
		botErr := NewBotError(ErrorTypeConfiguration, "CHANNEL_ID環境変数が設定されていません", nil)
		LogBotError(botErr)
		log.Fatal("CHANNEL_ID must be set in the .env file")
	}
	botToken := os.Getenv("TOKEN")
	if botToken == "" {
		botErr := NewBotError(ErrorTypeConfiguration, "TOKEN環境変数が設定されていません", nil)
		LogBotError(botErr)
		log.Fatal("TOKEN must be set in the .env file")
	}
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		botErr := NewBotError(ErrorTypeConfiguration, "GEMINI_API_KEY環境変数が設定されていません", nil)
		LogBotError(botErr)
		log.Fatal("GEMINI_API_KEY must be set in the .env file")
	}

	dumpFilePath := "/home/ubuntu/Bot/discord/yarikuri/dump_local_db/master_data_dump.sql"
	if err := loadMasterData(dumpFilePath); err != nil {
		if botErr, ok := err.(*BotError); ok {
			LogBotError(botErr)
		}
		log.Fatalf("マスターデータの読み込みに失敗しました: %v", err)
	}

	// 詳細説明サンプルを読み込み
	if err := loadDetailSamples(detailSamplesDir); err != nil {
		if botErr, ok := err.(*BotError); ok {
			LogBotError(botErr)
		}
		log.Fatalf("詳細説明サンプルの読み込みに失敗しました: %v", err)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiAPIKey))
	if err != nil {
		botErr := NewBotError(ErrorTypeAIService, "Gemini APIクライアントの初期化に失敗", err).
			WithContext("api_key_set", geminiAPIKey != "")
		LogBotError(botErr)
		log.Fatal(err)
	}
	geminiClient = client.GenerativeModel("gemini-1.5-flash-latest")
	log.Println("Gemini APIクライアントの初期化が完了しました。")

	transactions = make(map[string]*TransactionState)
	confirmationData = make(map[string]*ConfirmationData)
	masterDataQueues = make(map[string][]MasterQueueItem)

	// マスターキューファイルを読み込み
	err = loadMasterQueueFromFile()
	if err != nil {
		botErr := NewBotError(ErrorTypeFileIO, "マスターキューファイル読み込みエラー", err).
			WithContext("file_path", queueFilePath)
		LogBotError(botErr)
	}
	masterDataQueues = make(map[string][]MasterQueueItem)

	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		botErr := NewBotError(ErrorTypeDiscordAPI, "Discordセッション作成エラー", err).
			WithContext("bot_token_set", botToken != "")
		LogBotError(botErr)
		log.Fatalf("Error creating Discord session: %v", err)
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		log.Println("スラッシュコマンドを登録しています...")
		registeredCommands, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
		if err != nil {
			botErr := NewBotError(ErrorTypeDiscordAPI, "スラッシュコマンドの登録に失敗", err).
				WithContext("commands_count", len(commands))
			LogBotError(botErr)
		} else {
			log.Printf("%d個のコマンドを登録しました。", len(registeredCommands))
		}
	})
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			if strings.HasPrefix(customID, "paginate:") {
				handlePagination(s, i)
			} else if strings.HasPrefix(customID, "receipt_info_button:") {
				handleReceiptInfoButton(s, i)
			} else if strings.HasPrefix(customID, "category_select:") {
				handleCategorySelect(s, i)
			} else if strings.HasPrefix(customID, "category_search:") {
				handleCategorySearch(s, i)
			} else if strings.HasPrefix(customID, "edit_date:") {
				handleEditDate(s, i)
			} else if strings.HasPrefix(customID, "edit_amount:") {
				handleEditAmount(s, i)
			} else if strings.HasPrefix(customID, "edit_payment:") {
				handleEditPayment(s, i)
			} else if strings.HasPrefix(customID, "edit_group:") {
				handleEditGroup(s, i)
			} else if strings.HasPrefix(customID, "edit_payer:") {
				handleEditPayer(s, i)
			} else if strings.HasPrefix(customID, "edit_detail:") {
				handleEditDetail(s, i)
			} else if strings.HasPrefix(customID, "add_to_queue:") {
				handleAddToQueue(s, i)
			} else if strings.HasPrefix(customID, "cancel_entry:") {
				handleCancelEntry(s, i)
			} else if strings.HasPrefix(customID, "group_select:") {
				handleGroupSelect(s, i)
			} else if strings.HasPrefix(customID, "payer_select:") {
				handlePayerSelect(s, i)
			}
		case discordgo.InteractionModalSubmit:
			customID := i.ModalSubmitData().CustomID
			if strings.HasPrefix(customID, "receipt_info_modal:") {
				handleReceiptInfoModal(s, i)
			} else if strings.HasPrefix(customID, "category_search_modal:") {
				handleCategorySearchModal(s, i)
			} else if strings.HasPrefix(customID, "edit_date_modal:") {
				handleEditDateModal(s, i)
			} else if strings.HasPrefix(customID, "edit_amount_modal:") {
				handleEditAmountModal(s, i)
			} else if strings.HasPrefix(customID, "edit_payment_modal:") {
				handleEditPaymentModal(s, i)
			} else if strings.HasPrefix(customID, "edit_detail_modal:") {
				handleEditDetailModal(s, i)
			}
			log.Printf("モーダル送信を受信しました: %s", customID)
		}
	})

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil { log.Fatalf("Error opening connection: %v", err) }
	defer dg.Close()

	log.Println("Bot is now running. Press CTRL+C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

// =================================================================================
// 確認画面表示・更新関数
// =================================================================================

// updateConfirmationDisplay は確認画面を更新する
func updateConfirmationDisplay(s *discordgo.Session, messageID string) {
	data := getConfirmationData(messageID)
	if data == nil {
		log.Printf("確認データが見つかりません: %s", messageID)
		return
	}
	
	// カテゴリー名を取得
	var categoryName string = "不明"
	for _, category := range masterCategories {
		if category.ID == data.CategoryID {
			categoryName = category.Name
			break
		}
	}
	
	// グループ名を取得
	var groupName string = "なし"
	if data.GroupID != nil {
		for _, group := range masterGroups {
			if group.ID == *data.GroupID {
				groupName = group.Name
				break
			}
		}
	}
	
	// ユーザー名を取得
	var userName string = "不明"
	for _, user := range masterUsers {
		if user.ID == data.UserID {
			userName = user.Name
			break
		}
	}
	
	// Embedを作成（更新用）
	embed := &discordgo.MessageEmbed{
		Title: "📋 キューに追加前の確認 (更新済み)",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📅 日付", Value: data.Date, Inline: true},
			{Name: "💵 金額", Value: fmt.Sprintf("¥%d", data.Amount), Inline: true},
			{Name: "💳 支払い方法", Value: data.PaymentMethod, Inline: true},
			{Name: "📂 カテゴリー", Value: categoryName, Inline: true},
			{Name: "🏷️ グループ", Value: groupName, Inline: true},
			{Name: "👤 支払者", Value: userName, Inline: true},
			{Name: "📝 詳細", Value: data.Detail, Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "✅ データが更新されました。各項目を編集できます。問題なければ「キューに追加」をクリックしてください。",
		},
	}
	
	// 編集ボタンを作成
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_date:%s", messageID),
					Label:    "📅 日付を編集",
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_amount:%s", messageID),
					Label:    "💵 金額を編集",
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_payment:%s", messageID),
					Label:    "💳 支払い方法を編集",
					Style:    discordgo.SecondaryButton,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_group:%s", messageID),
					Label:    "🏷️ グループを編集",
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_payer:%s", messageID),
					Label:    "👤 支払者を編集",
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("edit_detail:%s", messageID),
					Label:    "📝 詳細を編集",
					Style:    discordgo.SecondaryButton,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("add_to_queue:%s", messageID),
					Label:    "✅ キューに追加",
					Style:    discordgo.SuccessButton,
				},
				discordgo.Button{
					CustomID: fmt.Sprintf("cancel_entry:%s", messageID),
					Label:    "❌ キャンセル",
					Style:    discordgo.DangerButton,
				},
			},
		},
	}
	
	// 新しいメッセージを送信（更新済み確認画面）
	_, err := s.ChannelMessageSendComplex(targetChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Printf("確認画面の更新に失敗: %v", err)
	} else {
		log.Printf("確認画面を更新しました: messageID=%s", messageID)
	}
}

// =================================================================================
// セレクトメニュー処理関数
// =================================================================================

// handleGroupSelect はグループ選択を処理する
func handleGroupSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "group_select:")
	
	selectedValue := i.MessageComponentData().Values[0]
	
	// データを更新
	updateConfirmationData(messageID, func(data *ConfirmationData) {
		if selectedValue == "none" {
			data.GroupID = nil
		} else {
			if groupID, err := strconv.Atoi(selectedValue); err == nil {
				data.GroupID = &groupID
			}
		}
	})
	
	// グループ名を取得
	var groupName string = "なし"
	if selectedValue != "none" {
		if groupID, err := strconv.Atoi(selectedValue); err == nil {
			for _, group := range masterGroups {
				if group.ID == groupID {
					groupName = group.Name
					break
				}
			}
		}
	}
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ グループを「%s」に更新しました。", groupName),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("グループ選択応答エラー: %v", err)
	}
	
	// 確認画面を更新
	updateConfirmationDisplay(s, messageID)
}

// handlePayerSelect は支払者選択を処理する
func handlePayerSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "payer_select:")
	
	selectedValue := i.MessageComponentData().Values[0]
	
	// データを更新
	if userID, err := strconv.Atoi(selectedValue); err == nil {
		updateConfirmationData(messageID, func(data *ConfirmationData) {
			data.UserID = userID
		})
	}
	
	// ユーザー名を取得
	var userName string = "不明"
	if userID, err := strconv.Atoi(selectedValue); err == nil {
		for _, user := range masterUsers {
			if user.ID == userID {
				userName = user.Name
				break
			}
		}
	}
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ 支払者を「%s」に更新しました。", userName),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("支払者選択応答エラー: %v", err)
	}
	
	// 確認画面を更新
	updateConfirmationDisplay(s, messageID)
}

// =================================================================================
// キュー操作関数
// =================================================================================

// handleAddToQueue はキューへの追加を処理する
func handleAddToQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "add_to_queue:")
	
	data := getConfirmationData(messageID)
	if data == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ エラー: データが見つかりません。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	// Expenseデータを作成
	expense := Expense{
		Date:       data.Date,
		Price:      data.Amount,
		CategoryID: data.CategoryID,
		UserID:     data.UserID,
		Detail:     data.Detail,
		GroupID:    data.GroupID,
	}
	
	// Expenseキューファイルに保存
	err := saveExpenseToQueue(expense)
	if err != nil {
		botErr := NewBotError(ErrorTypeFileIO, "Expenseキューファイル保存エラー", err).
			WithContext("expense", fmt.Sprintf("%+v", expense))
		LogBotError(botErr)
		
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ エラー: キューへの保存に失敗しました。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	
	log.Printf("キューに追加: %+v", expense)
	
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ データをキューに追加しました。",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		botErr := NewBotError(ErrorTypeDiscordAPI, "キュー追加応答エラー", err).
			WithContext("message_id", messageID)
		LogBotError(botErr)
	}

	// 確認データを削除
	mu.Lock()
	delete(confirmationData, messageID)
	mu.Unlock()
	
	log.Printf("キュー追加完了: messageID=%s", messageID)
}

// saveExpenseToQueue はExpenseをキューファイルに保存する
func saveExpenseToQueue(expense Expense) error {
	const expenseQueueFile = "../queues/expense_queue.json"
	
	// 既存のExpenseキューを読み込み
	var expenseQueue []Expense
	data, err := os.ReadFile(expenseQueueFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return NewBotError(ErrorTypeFileIO, "Expenseキューファイル読み込みエラー", err).
				WithContext("file_path", expenseQueueFile)
		}
		// ファイルが存在しない場合は空のキューで開始
		expenseQueue = []Expense{}
	} else {
		// 既存データをパース
		if err := json.Unmarshal(data, &expenseQueue); err != nil {
			return NewBotError(ErrorTypeFileIO, "ExpenseキューJSONパースエラー", err).
				WithContext("file_path", expenseQueueFile)
		}
	}
	
	// 新しいExpenseを追加
	expenseQueue = append(expenseQueue, expense)
	
	// ファイルに保存
	updatedData, err := json.MarshalIndent(expenseQueue, "", "  ")
	if err != nil {
		return NewBotError(ErrorTypeFileIO, "ExpenseキューJSON生成エラー", err).
			WithContext("queue_length", len(expenseQueue))
	}
	
	err = os.WriteFile(expenseQueueFile, updatedData, 0644)
	if err != nil {
		return NewBotError(ErrorTypeFileIO, "Expenseキューファイル書き込みエラー", err).
			WithContext("file_path", expenseQueueFile)
	}
	
	log.Printf("Expenseキューに追加完了: %s (total: %d件)", expenseQueueFile, len(expenseQueue))
	return nil
}

// getMasterDataWithQueue は既存マスターデータ + キューを結合して返す
func getMasterDataWithQueue(masterType string) interface{} {
		masterQueueMutex.RLock()
		queueItems := masterDataQueues[masterType]
		masterQueueMutex.RUnlock()
		
		switch masterType {
		case "category":
			result := make([]Category, len(masterCategories))
			copy(result, masterCategories)
			
			// キューからpendingアイテムを追加
			nextID := getNextCategoryID()
			for _, item := range queueItems {
				if item.Status == "pending" {
					result = append(result, Category{
						ID:   nextID,
						Name: item.Name,
					})
					nextID++
				}
			}
			
			// ソート
			sort.Slice(result, func(i, j int) bool {
				return sortJapaneseFirst(result[i].Name, result[j].Name)
			})
			
			return result
			
		case "group":
			result := make([]Group, len(masterGroups))
			copy(result, masterGroups)
			
			// キューからpendingアイテムを追加
			nextID := getNextGroupID()
			for _, item := range queueItems {
				if item.Status == "pending" {
					result = append(result, Group{
						ID:   nextID,
						Name: item.Name,
					})
					nextID++
				}
			}
			
			// ソート
			sort.Slice(result, func(i, j int) bool {
				return sortJapaneseFirst(result[i].Name, result[j].Name)
			})
			
			return result
			
		case "user":
			result := make([]User, len(masterUsers))
			copy(result, masterUsers)
			
			// キューからpendingアイテムを追加
			nextID := getNextUserID()
			for _, item := range queueItems {
				if item.Status == "pending" {
					result = append(result, User{
						ID:   nextID,
						Name: item.Name,
					})
					nextID++
				}
			}
			
			// ソート
			sort.Slice(result, func(i, j int) bool {
				return sortJapaneseFirst(result[i].Name, result[j].Name)
			})
			
			return result
			
		case "payment_type":
			result := make([]PaymentType, len(masterPaymentTypes))
			copy(result, masterPaymentTypes)
			
			// キューからpendingアイテムを追加
			nextID := getNextPaymentID()
			for _, item := range queueItems {
				if item.Status == "pending" {
					result = append(result, PaymentType{
						PayID:   nextID,
						PayKind: item.Name,
						TypeID:  item.TypeID,
					})
					nextID++
				}
			}
			
			// ソート
			sort.Slice(result, func(i, j int) bool {
				return sortJapaneseFirst(result[i].PayKind, result[j].PayKind)
			})
			
			return result
		}
		
		return nil
	}

// getNextCategoryID は次のカテゴリIDを取得する
func getNextCategoryID() int {
	maxID := 0
	for _, category := range masterCategories {
		if category.ID > maxID {
			maxID = category.ID
		}
	}
	return maxID + 1
}

// getNextGroupID は次のグループIDを取得する
func getNextGroupID() int {
	maxID := 0
	for _, group := range masterGroups {
		if group.ID > maxID {
			maxID = group.ID
		}
	}
	return maxID + 1
}

// getNextUserID は次のユーザーIDを取得する
func getNextUserID() int {
	maxID := 0
	for _, user := range masterUsers {
		if user.ID > maxID {
			maxID = user.ID
		}
	}
	return maxID + 1
}

// getNextPaymentID は次の支払いIDを取得する
func getNextPaymentID() int {
	maxID := 0
	for _, payment := range masterPaymentTypes {
		if payment.PayID > maxID {
			maxID = payment.PayID
		}
	}
	return maxID + 1
}

// =================================================================================
// マスターデータ追加機能
// =================================================================================

// handleAddMaster は新しいマスターデータの追加を処理する
func handleAddMaster(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options
		masterType := options[0].StringValue()
		name := options[1].StringValue()
		
		var typeName string
		if len(options) > 2 && options[2].StringValue() != "" {
			typeName = options[2].StringValue()
		}
		
		// バリデーション
		if name == "" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ データ名を入力してください。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		
		// 支払い方法の場合、TypeNameが必要
		if masterType == "payment_type" && typeName == "" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ 支払い方法の場合は、支払い種別を入力してください。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		
		// 重複チェック（既存マスター + キュー内）
		if isDuplicateMasterData(masterType, name) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ 「%s」は既に存在しています。", name),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		
		// TypeName バリデーション（支払い方法の場合）
		var typeID string
		if masterType == "payment_type" && typeName != "" {
			typeID = findTypeIDByName(typeName)
			if typeID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("❌ 支払い種別「%s」が見つかりません。", typeName),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
		}
		
		// キューに追加
		queueItem := MasterQueueItem{
			ID:        generateUniqueID(),
			Type:      masterType,
			Name:      name,
			TypeName:  typeName,
			TypeID:    typeID,
			Status:    "pending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err := addToMasterQueue(queueItem)
		if err != nil {
			log.Printf("マスターデータキューへの追加エラー: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ データの追加に失敗しました。",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		
		// 成功応答
		successMsg := fmt.Sprintf("✅ %s「%s」をキューに追加しました。", getMasterTypeName(masterType), name)
		if masterType == "payment_type" && typeName != "" {
			successMsg = fmt.Sprintf("✅ %s「%s」（種別：%s）をキューに追加しました。", getMasterTypeName(masterType), name, typeName)
		}
		successMsg += "次回同期時にマスターデータに反映されます。"
		
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: successMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	// isDuplicateMasterData は重複チェックを行う（既存マスター + キュー内）
	func isDuplicateMasterData(masterType, name string) bool {
		// 既存マスターデータをチェック
		switch masterType {
		case "category":
			for _, item := range masterCategories {
				if item.Name == name {
					return true
				}
			}
		case "group":
			for _, item := range masterGroups {
				if item.Name == name {
					return true
				}
			}
		case "user":
			for _, item := range masterUsers {
				if item.Name == name {
					return true
				}
			}
		case "payment_type":
			for _, item := range masterPaymentTypes {
				if item.PayKind == name {
					return true
				}
			}
		}
		
		// キュー内データもチェック
		masterQueueMutex.RLock()
		defer masterQueueMutex.RUnlock()
		
		if queueItems, exists := masterDataQueues[masterType]; exists {
			for _, item := range queueItems {
				if item.Name == name && item.Status != "error" {
					return true
				}
			}
		}
		
		return false
	}

	// findTypeIDByName はTypeNameからTypeIDを検索する
	func findTypeIDByName(typeName string) string {
		for _, item := range masterTypeList {
			if item.TypeName == typeName {
				return item.ID
			}
		}
		return ""
	}

	// addToMasterQueue はマスターデータをキューに追加する
	func addToMasterQueue(item MasterQueueItem) error {
		masterQueueMutex.Lock()
		defer masterQueueMutex.Unlock()
		
		// キューファイルを読み込み
		queueFilePath := "master_queue.json"
		var queues map[string][]MasterQueueItem
		
		data, err := os.ReadFile(queueFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				queues = make(map[string][]MasterQueueItem)
			} else {
				return err
			}
		} else {
			err = json.Unmarshal(data, &queues)
			if err != nil {
				queues = make(map[string][]MasterQueueItem)
			}
		}
		
		// キューに追加
		if queues[item.Type] == nil {
			queues[item.Type] = []MasterQueueItem{}
		}
		queues[item.Type] = append(queues[item.Type], item)
		
		// ファイルに保存
		updatedData, err := json.MarshalIndent(queues, "", "  ")
		if err != nil {
			return err
		}
		
		err = os.WriteFile(queueFilePath, updatedData, 0644)
		if err != nil {
			return err
		}
		
		// メモリ内キューも更新
		if masterDataQueues == nil {
			masterDataQueues = make(map[string][]MasterQueueItem)
		}
		masterDataQueues[item.Type] = queues[item.Type]
		
		return nil
	}

	// generateUniqueID は一意識別子を生成する
	func generateUniqueID() string {
		return fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Int63())
	}

	// getMasterTypeName はマスタータイプの日本語名を取得する
	func getMasterTypeName(masterType string) string {
		switch masterType {
		case "category":
			return "カテゴリ"
		case "group":
			return "グループ"
		case "user":
			return "ユーザー"
		case "payment_type":
			return "支払い方法"
		default:
			return "不明"
		}
	}

	// loadMasterQueueFromFile は起動時にキューファイルを読み込む
	func loadMasterQueueFromFile() error {
		queueFilePath := "master_queue.json"
		data, err := os.ReadFile(queueFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				masterDataQueues = make(map[string][]MasterQueueItem)
				return nil
			}
			return err
		}
		
		err = json.Unmarshal(data, &masterDataQueues)
		if err != nil {
			masterDataQueues = make(map[string][]MasterQueueItem)
			return err
		}
		
		return nil
	}

	
// handleCancelEntry はエントリのキャンセルを処理する
func handleCancelEntry(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	messageID := strings.TrimPrefix(customID, "cancel_entry:")
	
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ データの追加をキャンセルしました。",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("キャンセル応答エラー: %v", err)
	}
	
	// 確認データを削除
	mu.Lock()
	delete(confirmationData, messageID)
	mu.Unlock()
	
	log.Printf("エントリキャンセル: messageID=%s", messageID)
}
