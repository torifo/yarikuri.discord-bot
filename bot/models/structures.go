package models

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// Master data structures
type Category struct {
	ID   int
	Name string
}

type Group struct {
	ID   int
	Name string
}

type PaymentType struct {
	PayID    int
	PayKind  string
	TypeID   string
}

type User struct {
	ID   int
	Name string
}

type SourceList struct {
	ID         int
	SourceName string
	TypeID     int
}

type TypeKind struct {
	ID       int
	TypeName string
}

type TypeList struct {
	ID       string
	TypeName string
}

// Main data structures
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
	MessageID        string
	Date             string
	Amount           int
	CategoryID       int
	GroupID          *int
	UserID           int
	Detail           string
	PaymentMethod    string
	AIResult         ReceiptAnalysis
	OriginalAmount   *int    // 元の総額（分割処理用）
	RemainingAmount  *int    // 残り金額（分割処理用）
	IsPartialEntry   bool    // 分割エントリかどうか
	ParentMessageID  *string // 親のメッセージID（分割の場合）
}

// Master data queue item
type MasterQueueItem struct {
	ID        string    `json:"id"`              // 一意識別子
	Type      string    `json:"type"`            // category, group, user, payment_type
	Name      string    `json:"name"`            // 追加するデータ名
	TypeName  string    `json:"type_name,omitempty"` // 支払い方法の場合のTypeName
	TypeID    string    `json:"type_id,omitempty"`   // 支払い方法の場合のTypeID (自動計算)
	Status    string    `json:"status"`          // pending, synced, error
	CreatedAt time.Time `json:"created_at"`      // 作成日時
	UpdatedAt time.Time `json:"updated_at"`      // 更新日時
}