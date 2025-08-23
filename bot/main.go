// /home/ubuntu/Bot/discord/yarikuri/bot/main.go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

// =================================================================================
// グローバル変数定義
// =================================================================================
var (
	targetChannelID string
	geminiClient    *genai.GenerativeModel
	typeListMap     map[string]string
	typeKindMap     map[int]string
	transactions    map[string]*TransactionState // 進行中のトランザクションを管理

	masterCategories   []Category
	masterGroups       []Group
	masterPaymentTypes []PaymentType
	masterUsers        []User
	masterSourceList   []SourceList
	masterTypeKind     []TypeKind
	masterTypeList     []TypeList
)

const itemsPerPage = 15
const queueFilePath = "queue.json"
const tempImageDir = "img" // 一時画像保存用ディレクトリ

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

// ローカルAPIと共有するデータ構造
type Expense struct {
	Date       string `json:"date"`
	Price      int    `json:"price"`
	CategoryID int    `json:"category_id"`
	UserID     int    `json:"user_id"`
	Detail     string `json:"detail"`
	GroupID    *int   `json:"group_id,omitempty"`
	PaymentID  *int   `json:"payment_id,omitempty"`
}

// レシート解析のJSON出力に対応する構造体
type ReceiptAnalysis struct {
	IsReceipt     bool    `json:"is_receipt"`
	StoreName     *string `json:"store_name"`
	Date          *string `json:"date"`
	TotalAmount   *int    `json:"total_amount"`
	PaymentMethod *string `json:"payment_method"`
}

// 各トランザクションの状態を管理する構造体
type TransactionState struct {
	InitialMessageID string
	ImagePath        string
	AnalysisResult   ReceiptAnalysis
	// ... 今後ユーザーからの入力データを追加 ...
}


// =================================================================================
// データ読み込み・解析関連
// =================================================================================
func loadMasterData(filePath string) error {
	log.Println("マスターデータのダンプファイルを読み込んでいます...")
	sqlBytes, err := os.ReadFile(filePath)
	if err != nil { return err }
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
}

var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"check_master": handleCheckMaster,
	"show_master":  handleShowMaster,
	"add":          handleAdd,
	"fix":          handleFix,
}

// =================================================================================
// Discordイベントハンドラ
// =================================================================================

// messageCreate は、監視対象チャンネルにメッセージが投稿された時に呼び出される
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.ChannelID != targetChannelID || len(m.Attachments) == 0 {
		return
	}
	attachment := m.Attachments[0]
	if !strings.HasPrefix(attachment.ContentType, "image/") {
		return
	}

	log.Printf("画像を受信しました: %s (from: %s)", attachment.URL, m.Author.Username)
	
	msg, _ := s.ChannelMessageSend(m.ChannelID, "画像を受け付けました。ダウンロードと解析を開始します...")

	// --- 並行処理を開始 ---
	go processReceipt(s, m, msg)
}

// processReceipt は画像を処理してレシート解析を実行する
func processReceipt(s *discordgo.Session, m *discordgo.MessageCreate, msg *discordgo.Message) {
	attachment := m.Attachments[0]
	
	// 画像をダウンロード
	imagePath, err := downloadImage(attachment.URL)
	if err != nil {
		log.Printf("画像ダウンロードエラー: %v", err)
		s.ChannelMessageEdit(m.ChannelID, msg.ID, "画像のダウンロードに失敗しました。")
		return
	}
	
	// ダウンロード完了をユーザーに通知
	s.ChannelMessageEdit(m.ChannelID, msg.ID, "画像をダウンロードしました。現在解析中です...")
	
	// TODO: ここでGemini APIを使用してレシート解析を実行
	// 現在は簡単な応答のみ
	finalMessage := fmt.Sprintf("レシート画像を受信しました。\n画像パス: %s\n今後の実装で解析機能を追加予定です。", imagePath)
	s.ChannelMessageEdit(m.ChannelID, msg.ID, finalMessage)
}

// (handleCheckMaster, handleShowMaster は変更なし)
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
	if err != nil { log.Printf("ページデータ生成エラー: %v", err); return }
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{ Embeds: []*discordgo.MessageEmbed{embed}, Components: components, },
	})
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

// handleFix は /fix コマンドの処理
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

// handlePagination は、ページネーションボタンが押された時の処理
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


// =================================================================================
// ヘルパー関数
// =================================================================================
func generatePaginatedData(dataType string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	var allItems []string
	var title string
	switch dataType {
	case "category":
		title = "カテゴリ一覧"
		for _, item := range masterCategories { allItems = append(allItems, item.Name) }
	case "group":
		title = "グループ一覧"
		for _, item := range masterGroups { allItems = append(allItems, item.Name) }
	case "user":
		title = "ユーザー一覧"
		for _, item := range masterUsers { allItems = append(allItems, item.Name) }
	case "payment_type":
		title = "支払い方法一覧"
		for _, item := range masterPaymentTypes {
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
	if err != nil { return "", err }
	defer response.Body.Close()

	os.MkdirAll(tempImageDir, 0755)

	filePath := filepath.Join(tempImageDir, filepath.Base(response.Request.URL.Path))
	file, err := os.Create(filePath)
	if err != nil { return "", err }
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil { return "", err }

	return filePath, nil
}

// =================================================================================
// main関数 (Botの起動)
// =================================================================================
func main() {
	err := godotenv.Load()
	if err != nil { log.Println("Note: .env file not found, continuing without it.") }

	targetChannelID = os.Getenv("CHANNEL_ID")
	if targetChannelID == "" { log.Fatal("CHANNEL_ID must be set in the .env file") }
	botToken := os.Getenv("TOKEN")
	if botToken == "" { log.Fatal("TOKEN must be set in the .env file") }
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" { log.Fatal("GEMINI_API_KEY must be set in the .env file") }

	dumpFilePath := "/home/ubuntu/Bot/discord/yarikuri/dump_local_db/master_data_dump.sql"
	if err := loadMasterData(dumpFilePath); err != nil {
		log.Fatalf("マスターデータの読み込みに失敗しました: %v", err)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiAPIKey))
	if err != nil { log.Fatal(err) }
	geminiClient = client.GenerativeModel("gemini-1.5-flash-latest")
	log.Println("Gemini APIクライアントの初期化が完了しました。")

	transactions = make(map[string]*TransactionState)

	dg, err := discordgo.New("Bot " + botToken)
	if err != nil { log.Fatalf("Error creating Discord session: %v", err) }

	dg.AddHandler(messageCreate)
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		log.Println("スラッシュコマンドを登録しています...")
		registeredCommands, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
		if err != nil { log.Printf("コマンドの登録に失敗しました: %v", err) } else { log.Printf("%d個のコマンドを登録しました。", len(registeredCommands)) }
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
			}
		// (モーダル送信時の処理を追加)
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
