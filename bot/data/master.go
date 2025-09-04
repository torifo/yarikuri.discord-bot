package data

import (
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/yarikuri/errors"
	"github.com/yarikuri/models"
	"github.com/yarikuri/utils"
)

// LoadMasterData はマスターデータをSQLダンプファイルから読み込む
func LoadMasterData(filePath string, state *models.BotState) error {
	log.Println("マスターデータのダンプファイルを読み込んでいます...")
	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		botErr := errors.NewBotError(errors.ErrorTypeFileIO, "マスターデータダンプファイルの読み込みに失敗", err).
			WithContext("file_path", filePath)
		return botErr
	}
	sqlContent := string(sqlBytes)

	// カテゴリリストの読み込み
	records, _ := parseTableData(sqlContent, "category_list")
	var categories []models.Category
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		categories = append(categories, models.Category{
			ID:   id,
			Name: strings.TrimSpace(rec[1]),
		})
	}
	sort.Slice(categories, func(i, j int) bool {
		return utils.SortJapaneseFirst(categories[i].Name, categories[j].Name)
	})
	state.SetMasterCategories(categories)
	log.Printf("-> %d件のカテゴリを読み込み、ソートしました。\n", len(categories))

	// グループリストの読み込み
	records, _ = parseTableData(sqlContent, "group_list")
	var groups []models.Group
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		groups = append(groups, models.Group{
			ID:   id,
			Name: strings.TrimSpace(rec[1]),
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return utils.SortJapaneseFirst(groups[i].Name, groups[j].Name)
	})
	state.SetMasterGroups(groups)
	log.Printf("-> %d件のグループを読み込み、ソートしました。\n", len(groups))

	// 支払い方法の読み込み
	records, _ = parseTableData(sqlContent, "payment_type")
	var paymentTypes []models.PaymentType
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		paymentTypes = append(paymentTypes, models.PaymentType{
			PayID:   id,
			PayKind: strings.TrimSpace(rec[1]),
			TypeID:  strings.TrimSpace(rec[2]),
		})
	}
	sort.Slice(paymentTypes, func(i, j int) bool {
		return utils.SortJapaneseFirst(paymentTypes[i].PayKind, paymentTypes[j].PayKind)
	})
	state.SetMasterPaymentTypes(paymentTypes)
	log.Printf("-> %d件の支払い方法を読み込み、ソートしました。\n", len(paymentTypes))

	// ユーザーリストの読み込み
	records, _ = parseTableData(sqlContent, "user_list")
	var users []models.User
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		users = append(users, models.User{
			ID:   id,
			Name: strings.TrimSpace(rec[1]),
		})
	}
	sort.Slice(users, func(i, j int) bool {
		return utils.SortJapaneseFirst(users[i].Name, users[j].Name)
	})
	state.SetMasterUsers(users)
	log.Printf("-> %d件のユーザーを読み込み、ソートしました。\n", len(users))

	// 収入源リストの読み込み
	records, _ = parseTableData(sqlContent, "source_list")
	var sourceList []models.SourceList
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		typeId, _ := strconv.Atoi(strings.TrimSpace(rec[2]))
		sourceList = append(sourceList, models.SourceList{
			ID:         id,
			SourceName: strings.TrimSpace(rec[1]),
			TypeID:     typeId,
		})
	}
	sort.Slice(sourceList, func(i, j int) bool {
		return utils.SortJapaneseFirst(sourceList[i].SourceName, sourceList[j].SourceName)
	})
	state.SetMasterSourceList(sourceList)
	log.Printf("-> %d件の収入源を読み込み、ソートしました。\n", len(sourceList))

	// 収入種別の読み込み
	records, _ = parseTableData(sqlContent, "type_kind")
	var typeKind []models.TypeKind
	for _, rec := range records {
		id, _ := strconv.Atoi(strings.TrimSpace(rec[0]))
		typeKind = append(typeKind, models.TypeKind{
			ID:       id,
			TypeName: strings.TrimSpace(rec[1]),
		})
	}
	state.SetMasterTypeKind(typeKind)

	// TypeKindMapの作成
	state.TypeKindMap = make(map[int]string)
	for _, item := range typeKind {
		state.TypeKindMap[item.ID] = item.TypeName
	}
	log.Printf("-> %d件の収入種別を読み込み、マップを作成しました。\n", len(typeKind))

	// タイプリストの読み込み
	records, _ = parseTableData(sqlContent, "type_list")
	var typeList []models.TypeList
	for _, rec := range records {
		typeList = append(typeList, models.TypeList{
			ID:       strings.TrimSpace(rec[0]),
			TypeName: strings.TrimSpace(rec[1]),
		})
	}
	state.SetMasterTypeList(typeList)

	// TypeListMapの作成
	state.TypeListMap = make(map[string]string)
	for _, item := range typeList {
		state.TypeListMap[item.ID] = item.TypeName
	}
	log.Printf("-> %d件のタイプリストを読み込み、マップを作成しました。\n", len(typeList))

	log.Println("全てのマスターデータの読み込みが完了しました。")
	return nil
}

// parseTableData はSQLダンプから指定されたテーブルのデータを抽出する
func parseTableData(sqlContent, tableName string) ([][]string, error) {
	startMarker := "COPY public." + tableName
	endMarker := "\\."
	startIndex := strings.Index(sqlContent, startMarker)
	if startIndex == -1 {
		return nil, nil
	}
	dataStartIndex := strings.Index(sqlContent[startIndex:], ";")
	if dataStartIndex == -1 {
		return nil, nil
	}
	dataBlockStartIndex := startIndex + dataStartIndex + 1
	endIndex := strings.Index(sqlContent[dataBlockStartIndex:], endMarker)
	if endIndex == -1 {
		return nil, nil
	}
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