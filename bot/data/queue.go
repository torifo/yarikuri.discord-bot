package data

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/yarikuri/errors"
	"github.com/yarikuri/models"
)

// QueueManager はマスターデータキューを管理する
type QueueManager struct {
	mutex           sync.RWMutex
	queueFilePath   string
	masterDataQueue map[string][]models.MasterQueueItem
}

// NewQueueManager は新しいキューマネージャを作成
func NewQueueManager(queueFilePath string) *QueueManager {
	return &QueueManager{
		queueFilePath:   queueFilePath,
		masterDataQueue: make(map[string][]models.MasterQueueItem),
	}
}

// LoadFromFile はキューファイルから読み込む
func (qm *QueueManager) LoadFromFile() error {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	data, err := os.ReadFile(qm.queueFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			qm.masterDataQueue = make(map[string][]models.MasterQueueItem)
			return nil
		}
		return errors.NewBotError(errors.ErrorTypeFileIO, "キューファイル読み込みエラー", err).
			WithContext("file_path", qm.queueFilePath)
	}

	err = json.Unmarshal(data, &qm.masterDataQueue)
	if err != nil {
		qm.masterDataQueue = make(map[string][]models.MasterQueueItem)
		return errors.NewBotError(errors.ErrorTypeDataAccess, "キューデータ解析エラー", err).
			WithContext("file_path", qm.queueFilePath)
	}

	return nil
}

// SaveToFile はキューをファイルに保存
func (qm *QueueManager) SaveToFile() error {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()

	updatedData, err := json.MarshalIndent(qm.masterDataQueue, "", "  ")
	if err != nil {
		return errors.NewBotError(errors.ErrorTypeDataAccess, "キューデータ変換エラー", err)
	}

	err = os.WriteFile(qm.queueFilePath, updatedData, 0644)
	if err != nil {
		return errors.NewBotError(errors.ErrorTypeFileIO, "キューファイル保存エラー", err).
			WithContext("file_path", qm.queueFilePath)
	}

	return nil
}

// AddToQueue はアイテムをキューに追加
func (qm *QueueManager) AddToQueue(item models.MasterQueueItem) error {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	// キューに追加
	if qm.masterDataQueue[item.Type] == nil {
		qm.masterDataQueue[item.Type] = []models.MasterQueueItem{}
	}
	qm.masterDataQueue[item.Type] = append(qm.masterDataQueue[item.Type], item)

	// ファイルに保存
	return qm.saveToFileUnsafe()
}

// GetQueue は指定されたタイプのキューを取得
func (qm *QueueManager) GetQueue(queueType string) []models.MasterQueueItem {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()

	queue, exists := qm.masterDataQueue[queueType]
	if !exists {
		return []models.MasterQueueItem{}
	}

	// コピーを返す
	result := make([]models.MasterQueueItem, len(queue))
	copy(result, queue)
	return result
}

// GetAllQueues は全キューを取得
func (qm *QueueManager) GetAllQueues() map[string][]models.MasterQueueItem {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()

	// 深いコピーを作成
	result := make(map[string][]models.MasterQueueItem)
	for queueType, queue := range qm.masterDataQueue {
		result[queueType] = make([]models.MasterQueueItem, len(queue))
		copy(result[queueType], queue)
	}

	return result
}

// UpdateItemStatus はアイテムのステータスを更新
func (qm *QueueManager) UpdateItemStatus(itemID, status string) error {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	for queueType, queue := range qm.masterDataQueue {
		for i, item := range queue {
			if item.ID == itemID {
				qm.masterDataQueue[queueType][i].Status = status
				qm.masterDataQueue[queueType][i].UpdatedAt = time.Now()
				return qm.saveToFileUnsafe()
			}
		}
	}

	return errors.NewBotError(errors.ErrorTypeDataAccess, "指定されたアイテムが見つかりません", nil).
		WithContext("item_id", itemID)
}

// RemoveItem はアイテムをキューから削除
func (qm *QueueManager) RemoveItem(itemID string) error {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	for queueType, queue := range qm.masterDataQueue {
		for i, item := range queue {
			if item.ID == itemID {
				// アイテムを削除
				qm.masterDataQueue[queueType] = append(queue[:i], queue[i+1:]...)
				return qm.saveToFileUnsafe()
			}
		}
	}

	return errors.NewBotError(errors.ErrorTypeDataAccess, "指定されたアイテムが見つかりません", nil).
		WithContext("item_id", itemID)
}

// saveToFileUnsafe はmutexがロックされている状態でファイル保存を実行
func (qm *QueueManager) saveToFileUnsafe() error {
	updatedData, err := json.MarshalIndent(qm.masterDataQueue, "", "  ")
	if err != nil {
		return errors.NewBotError(errors.ErrorTypeDataAccess, "キューデータ変換エラー", err)
	}

	err = os.WriteFile(qm.queueFilePath, updatedData, 0644)
	if err != nil {
		return errors.NewBotError(errors.ErrorTypeFileIO, "キューファイル保存エラー", err).
			WithContext("file_path", qm.queueFilePath)
	}

	return nil
}

// GenerateUniqueID は一意識別子を生成
func GenerateUniqueID() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Int63())
}

// LoadExpenseQueue は支出キューファイルから読み込む
func LoadExpenseQueue(queueFilePath string) ([]models.Expense, error) {
	var expenseQueue []models.Expense
	data, err := os.ReadFile(queueFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.NewBotError(errors.ErrorTypeFileIO, "Expenseキューファイル読み込みエラー", err).
				WithContext("file_path", queueFilePath)
		}
		// ファイルが存在しない場合は空のキューで開始
		return expenseQueue, nil
	}

	err = json.Unmarshal(data, &expenseQueue)
	if err != nil {
		return nil, errors.NewBotError(errors.ErrorTypeDataAccess, "Expenseキューデータ解析エラー", err).
			WithContext("file_path", queueFilePath)
	}

	return expenseQueue, nil
}