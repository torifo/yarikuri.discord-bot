package models

import (
	"sync"

	"github.com/google/generative-ai-go/genai"
)

// BotState は Bot のグローバル状態を管理する構造体
type BotState struct {
	mu sync.RWMutex

	// Discord 関連
	TargetChannelID string

	// AI 関連
	GeminiClient *genai.GenerativeModel

	// マップ系データ
	TypeListMap   map[string]string // TypeID -> TypeName
	TypeKindMap   map[int]string    // ID -> TypeName
	DetailSamples map[string]string // カテゴリ名 -> 詳細説明サンプル

	// 状態管理マップ
	Transactions     map[string]*TransactionState // 進行中のトランザクションを管理
	ConfirmationData map[string]*ConfirmationData // 確認画面のデータを管理

	// マスターデータ（読み取り専用）
	MasterCategories   []Category
	MasterGroups       []Group
	MasterPaymentTypes []PaymentType
	MasterUsers        []User
	MasterSourceList   []SourceList
	MasterTypeKind     []TypeKind
	MasterTypeList     []TypeList

	// キューデータ
	MasterDataQueues map[string][]MasterQueueItem
	QueueMutex       sync.RWMutex
}

// NewBotState は新しい BotState インスタンスを作成
func NewBotState() *BotState {
	return &BotState{
		TypeListMap:      make(map[string]string),
		TypeKindMap:      make(map[int]string),
		DetailSamples:    make(map[string]string),
		Transactions:     make(map[string]*TransactionState),
		ConfirmationData: make(map[string]*ConfirmationData),
		MasterDataQueues: make(map[string][]MasterQueueItem),
	}
}

// GetMasterData は指定された種別のマスターデータを返す
func (s *BotState) GetMasterData(dataType string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch dataType {
	case "categories":
		return s.MasterCategories
	case "groups":
		return s.MasterGroups
	case "payment_types":
		return s.MasterPaymentTypes
	case "users":
		return s.MasterUsers
	case "source_list":
		return s.MasterSourceList
	case "type_kind":
		return s.MasterTypeKind
	case "type_list":
		return s.MasterTypeList
	default:
		return nil
	}
}

// SetMasterCategories はマスターカテゴリデータを設定
func (s *BotState) SetMasterCategories(categories []Category) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MasterCategories = categories
}

// SetMasterGroups はマスターグループデータを設定
func (s *BotState) SetMasterGroups(groups []Group) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MasterGroups = groups
}

// SetMasterPaymentTypes はマスター支払い種別データを設定
func (s *BotState) SetMasterPaymentTypes(paymentTypes []PaymentType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MasterPaymentTypes = paymentTypes
}

// SetMasterUsers はマスターユーザデータを設定
func (s *BotState) SetMasterUsers(users []User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MasterUsers = users
}

// SetMasterSourceList はマスターソースリストデータを設定
func (s *BotState) SetMasterSourceList(sourceList []SourceList) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MasterSourceList = sourceList
}

// SetMasterTypeKind はマスタータイプ種別データを設定
func (s *BotState) SetMasterTypeKind(typeKind []TypeKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MasterTypeKind = typeKind
}

// SetMasterTypeList はマスタータイプリストデータを設定
func (s *BotState) SetMasterTypeList(typeList []TypeList) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MasterTypeList = typeList
}

// GetTransaction は指定されたキーのトランザクションを返す
func (s *BotState) GetTransaction(key string) (*TransactionState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	transaction, exists := s.Transactions[key]
	return transaction, exists
}

// SetTransaction はトランザクションを設定
func (s *BotState) SetTransaction(key string, transaction *TransactionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Transactions[key] = transaction
}

// DeleteTransaction はトランザクションを削除
func (s *BotState) DeleteTransaction(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Transactions, key)
}

// GetConfirmationData は指定されたキーの確認データを返す
func (s *BotState) GetConfirmationData(key string) (*ConfirmationData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, exists := s.ConfirmationData[key]
	return data, exists
}

// SetConfirmationData は確認データを設定
func (s *BotState) SetConfirmationData(key string, data *ConfirmationData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConfirmationData[key] = data
}

// DeleteConfirmationData は確認データを削除
func (s *BotState) DeleteConfirmationData(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ConfirmationData, key)
}