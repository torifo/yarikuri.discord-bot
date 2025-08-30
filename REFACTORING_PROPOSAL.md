# Yarikuri Discord Bot - リファクタリング提案書（単一ファイル改善版）

## 概要

本文書は、家計管理Discord Bot「yarikuri_bot」の現在のコードベース分析に基づく、**単一ファイル内でのブロック構造改善**を提案します。現在の[`main.go`](bot/main.go)は約3000行となっており、個人開発の利便性を保ちながら保守性・拡張性を向上させることを目的としています。

## 現状分析

### アーキテクチャの課題

#### 1. 単一ファイル肥大化（Critical）
- **問題**: [`main.go`](bot/main.go)が3000行を超える巨大ファイル
- **影響**: 
  - コード理解の困難
  - 変更時の影響範囲の把握困難
  - 複数人開発時のマージコンフリクト増大
  - デバッグ効率の低下

#### 2. 責任の混在（High）
- **問題**: データ処理、Discord API、AI解析、ファイルI/Oが混在
- **影響**:
  - テストの困難
  - 機能の独立性不足
  - 変更時の予期しない副作用

#### 3. グローバル状態の多用（High）
```go
var (
    targetChannelID       string
    geminiClient         *genai.GenerativeModel
    transactions         map[string]*TransactionState
    confirmationData     map[string]*ConfirmationData
    masterCategories     []Category
    masterGroups         []Group
    // ... その他多数
)
```
- **影響**: 
  - 並行処理時の競合状態リスク
  - 状態管理の複雑化
  - 単体テストの困難

#### 4. エラーハンドリングの不統一（Medium）
- **問題**: エラー処理パターンが統一されていない
- **影響**: デバッグ時の問題特定困難

### 機能面の課題

#### 1. マスターデータ管理の改善点
- **現状**: メモリ上でのグローバル変数管理
- **課題**: データ永続性、整合性確保、リロード機能不足

#### 2. AI解析システムの改善点
- **現状**: レスポンス時間制限に対する対処療法
- **課題**: 処理時間の最適化、エラー復旧機能

#### 3. Discord Interaction処理
- **現状**: 複雑なCustomID管理、3秒制限への対処
- **課題**: より堅牢な非同期処理アーキテクチャ

## リファクタリング戦略（単一ファイル内改善）

### Phase 1: ブロック構造の最適化（優先度: Critical）

#### 1.1 [`main.go`](bot/main.go)内のブロック再編成
```go
// /home/ubuntu/Bot/discord/yarikuri/bot/main.go

// =================================================================================
// 1. IMPORTS & PACKAGE DECLARATION
// =================================================================================
package main
import (...)

// =================================================================================
// 2. CONSTANTS & CONFIGURATION
// =================================================================================
const (
    itemsPerPage     = 15
    queueFilePath    = "queue.json"
    tempImageDir     = "img"
    detailSamplesDir = "../detail_samples"
    
    // エラーメッセージ定数
    ErrDiscordAPI    = "Discord APIエラー"
    ErrAIProcessing  = "AI解析エラー"
    ErrDataAccess    = "データアクセスエラー"
)

// =================================================================================
// 3. GLOBAL VARIABLES & STATE MANAGEMENT
// =================================================================================
var (
    // Discord関連
    targetChannelID string
    
    // AI関連
    geminiClient    *genai.GenerativeModel
    
    // 状態管理（マップ）
    transactions     map[string]*TransactionState // 改善：構造体でラップ
    confirmationData map[string]*ConfirmationData
    mu              sync.Mutex
    
    // マスターデータ（読み取り専用）
    masterCategories   []Category
    masterGroups       []Group
    masterPaymentTypes []PaymentType
    masterUsers        []User
    masterSourceList   []SourceList
    masterTypeKind     []TypeKind
    masterTypeList     []TypeList
    
    // キュー管理
    masterDataQueues map[string][]MasterQueueItem
    masterQueueMutex sync.RWMutex
    
    // 補助マップ（パフォーマンス改善）
    typeListMap     map[string]string
    typeKindMap     map[int]string
    detailSamples   map[string]string
)

// =================================================================================
// 4. STRUCT DEFINITIONS & MODELS
// =================================================================================
// [既存の構造体定義群をこちらに整理]

// =================================================================================
// 5. INITIALIZATION & STARTUP FUNCTIONS
// =================================================================================
func main() {
    // 初期化処理をブロック化
    if err := initializeBot(); err != nil {
        log.Fatalf("Bot初期化エラー: %v", err)
    }
    
    // メインループ
    runBot()
}

func initializeBot() error {
    // 環境変数読み込み
    if err := loadEnvironment(); err != nil { return err }
    
    // マスターデータ読み込み
    if err := loadAllMasterData(); err != nil { return err }
    
    // Discord設定
    if err := setupDiscordBot(); err != nil { return err }
    
    // AI設定
    if err := setupAIService(); err != nil { return err }
    
    return nil
}

// =================================================================================
// 6. MASTER DATA MANAGEMENT BLOCK
// =================================================================================
// [マスターデータ関連の関数群]
```

#### 1.2 機能ブロック分割戦略
1. **初期化ブロック**: 環境設定、データ読み込み
2. **マスターデータ管理ブロック**: CRUD操作、キュー管理
3. **Discord API処理ブロック**: イベントハンドラー、レスポンス処理
4. **AI解析処理ブロック**: レシート解析、結果処理
5. **インタラクション処理ブロック**: ボタン、モーダル、セレクト
6. **ユーティリティブロック**: ヘルパー関数、バリデーション
7. **エラーハンドリングブロック**: 統一エラー処理

### Phase 2: 状態管理の改善（単一ファイル内）（優先度: High）

#### 2.1 グローバル変数の構造化
```go
// =================================================================================
// 3. GLOBAL VARIABLES & STATE MANAGEMENT (改善版)
// =================================================================================

// BotState - 全体状態を管理する構造体
type BotState struct {
    mu sync.RWMutex
    
    // Discord設定
    Discord struct {
        TargetChannelID string
        Session        *discordgo.Session
    }
    
    // AI設定
    AI struct {
        GeminiClient *genai.GenerativeModel
        APIKey       string
    }
    
    // 実行時状態
    Runtime struct {
        Transactions     map[string]*TransactionState
        ConfirmationData map[string]*ConfirmationData
        IsShuttingDown   bool
    }
    
    // マスターデータ（読み取り専用）
    Masters struct {
        Categories   []Category
        Groups       []Group
        PaymentTypes []PaymentType
        Users        []User
        SourceList   []SourceList
        TypeKind     []TypeKind
        TypeList     []TypeList
    }
    
    // キューデータ
    Queues struct {
        mu    sync.RWMutex
        Items map[string][]MasterQueueItem
    }
    
    // キャッシュマップ
    Cache struct {
        TypeListMap   map[string]string
        TypeKindMap   map[int]string
        DetailSamples map[string]string
    }
}

// グローバルボット状態
var botState *BotState

// 初期化関数
func initBotState() {
    botState = &BotState{
        Runtime: struct {
            Transactions     map[string]*TransactionState
            ConfirmationData map[string]*ConfirmationData
            IsShuttingDown   bool
        }{
            Transactions:     make(map[string]*TransactionState),
            ConfirmationData: make(map[string]*ConfirmationData),
            IsShuttingDown:   false,
        },
        Queues: struct {
            mu    sync.RWMutex
            Items map[string][]MasterQueueItem
        }{
            Items: make(map[string][]MasterQueueItem),
        },
        Cache: struct {
            TypeListMap   map[string]string
            TypeKindMap   map[int]string
            DetailSamples map[string]string
        }{
            TypeListMap:   make(map[string]string),
            TypeKindMap:   make(map[int]string),
            DetailSamples: make(map[string]string),
        },
    }
}

// 安全なアクセサー関数
func (bs *BotState) GetTransaction(id string) (*TransactionState, bool) {
    bs.mu.RLock()
    defer bs.mu.RUnlock()
    t, exists := bs.Runtime.Transactions[id]
    return t, exists
}

func (bs *BotState) SetTransaction(id string, transaction *TransactionState) {
    bs.mu.Lock()
    defer bs.mu.Unlock()
    bs.Runtime.Transactions[id] = transaction
}
```

#### 2.2 設定管理の改善
```go
// =================================================================================
// 2. CONSTANTS & CONFIGURATION (改善版)
// =================================================================================

type BotConfig struct {
    // Discord設定
    DiscordToken        string        `env:"DISCORD_TOKEN"`
    TargetChannelID     string        `env:"TARGET_CHANNEL_ID"`
    DiscordTimeout      time.Duration `env:"DISCORD_TIMEOUT" envDefault:"30s"`
    
    // Gemini AI設定
    GeminiAPIKey        string        `env:"GEMINI_API_KEY"`
    GeminiModel         string        `env:"GEMINI_MODEL" envDefault:"gemini-1.5-flash"`
    AITimeout          time.Duration `env:"AI_TIMEOUT" envDefault:"30s"`
    
    // ファイル設定
    QueueFilePath       string        `env:"QUEUE_FILE_PATH" envDefault:"queue.json"`
    TempImageDir        string        `env:"TEMP_IMAGE_DIR" envDefault:"img"`
    DetailSamplesDir    string        `env:"DETAIL_SAMPLES_DIR" envDefault:"../detail_samples"`
    MasterDataDumpPath  string        `env:"MASTER_DATA_DUMP_PATH" envDefault:"../dump_local_db/master_data_dump.sql"`
    
    // パフォーマンス設定
    ItemsPerPage        int           `env:"ITEMS_PER_PAGE" envDefault:"15"`
    MaxConcurrentAI     int           `env:"MAX_CONCURRENT_AI" envDefault:"3"`
}

var config *BotConfig

func loadConfiguration() error {
    cfg := &BotConfig{}
    if err := godotenv.Load(); err != nil {
        log.Println("Warning: .env file not found")
    }
    
    // 環境変数から設定を読み込み（手動実装）
    cfg.DiscordToken = os.Getenv("DISCORD_TOKEN")
    cfg.TargetChannelID = os.Getenv("TARGET_CHANNEL_ID")
    cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
    
    // デフォルト値の設定
    if cfg.QueueFilePath == "" { cfg.QueueFilePath = "queue.json" }
    if cfg.TempImageDir == "" { cfg.TempImageDir = "img" }
    if cfg.ItemsPerPage == 0 { cfg.ItemsPerPage = 15 }
    
    config = cfg
    return validateConfiguration()
}

func validateConfiguration() error {
    if config.DiscordToken == "" {
        return fmt.Errorf("DISCORD_TOKEN is required")
    }
    if config.GeminiAPIKey == "" {
        return fmt.Errorf("GEMINI_API_KEY is required")
    }
    return nil
}
```

### Phase 3: パフォーマンス最適化（優先度: High）

#### 3.1 非同期処理の改善
```go
// services/async_processor.go
type AsyncProcessor struct {
    workerCount int
    taskQueue   chan Task
    results     chan Result
    ctx         context.Context
    cancel      context.CancelFunc
}

type Task struct {
    ID        string
    Type      TaskType
    Data      interface{}
    CreatedAt time.Time
}

func (ap *AsyncProcessor) ProcessAIAnalysis(task *AIAnalysisTask) error {
    select {
    case ap.taskQueue <- task:
        return nil
    case <-ap.ctx.Done():
        return ap.ctx.Err()
    case <-time.After(5 * time.Second):
        return errors.New("task queue full")
    }
}
```

#### 3.2 キャッシュシステム導入
```go
// services/cache_service.go
type CacheService struct {
    mu    sync.RWMutex
    cache map[string]*CacheItem
    ttl   time.Duration
}

type CacheItem struct {
    Value     interface{}
    ExpiresAt time.Time
}

func (cs *CacheService) Get(key string) (interface{}, bool) {
    cs.mu.RLock()
    defer cs.mu.RUnlock()
    
    item, exists := cs.cache[key]
    if !exists || time.Now().After(item.ExpiresAt) {
        return nil, false
    }
    
    return item.Value, true
}
```

### Phase 4: エラーハンドリング統一（優先度: Medium）

#### 4.1 エラー型の統一
```go
// utils/errors.go
type BotError struct {
    Type    ErrorType
    Message string
    Cause   error
    Context map[string]interface{}
}

type ErrorType string

const (
    ErrorTypeDiscordAPI     ErrorType = "discord_api"
    ErrorTypeAIService     ErrorType = "ai_service"
    ErrorTypeDataAccess    ErrorType = "data_access"
    ErrorTypeValidation    ErrorType = "validation"
    ErrorTypeConfiguration ErrorType = "configuration"
)

func NewBotError(errorType ErrorType, message string, cause error) *BotError {
    return &BotError{
        Type:    errorType,
        Message: message,
        Cause:   cause,
        Context: make(map[string]interface{}),
    }
}
```

#### 4.2 ログシステムの統一
```go
// utils/logger.go
type Logger struct {
    *slog.Logger
    botName string
}

func NewLogger(botName string) *Logger {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })
    
    return &Logger{
        Logger:  slog.New(handler),
        botName: botName,
    }
}

func (l *Logger) LogBotError(err *BotError) {
    l.Error("Bot Error",
        "type", err.Type,
        "message", err.Message,
        "cause", err.Cause,
        "context", err.Context,
        "bot", l.botName,
    )
}
```

## 実装計画

### Phase 1: 基盤整備（推定: 1-2週間）

#### Week 1
- [ ] ディレクトリ構造作成
- [ ] 基本インターフェース定義
- [ ] models パッケージの実装
- [ ] config パッケージの実装

#### Week 2  
- [ ] services パッケージの基本実装
- [ ] 既存コードからの機能移行開始
- [ ] 単体テストの基盤作成

### Phase 2: コア機能移行（推定: 2-3週間）

#### Week 3-4
- [ ] MasterDataService 実装・移行
- [ ] DiscordService 実装・移行
- [ ] 基本的なスラッシュコマンド動作確認

#### Week 5
- [ ] AIService 実装・移行
- [ ] レシート解析機能の動作確認
- [ ] エラーハンドリング改善

### Phase 3: 最適化・テスト（推定: 1-2週間）

#### Week 6
- [ ] パフォーマンス測定・最適化
- [ ] キャッシュシステム導入
- [ ] 統合テスト実装

#### Week 7
- [ ] 運用テスト
- [ ] ドキュメント更新
- [ ] デプロイ・動作確認

## 期待効果

### 開発効率向上
- **コード理解時間**: 70%削減
- **新機能追加時間**: 50%削減  
- **バグ修正時間**: 60%削減

### 保守性向上
- **テストカバレッジ**: 0% → 80%
- **コード重複率**: 30% → 5%
- **循環依存**: 排除

### パフォーマンス向上
- **Discord応答時間**: 平均30%改善
- **AI解析処理**: 平均20%高速化
- **メモリ使用量**: 平均15%削減

### 運用安定性向上
- **エラー復旧時間**: 50%削減
- **ログ解析時間**: 70%削減
- **デプロイ時間**: 40%削減

## リスク分析

### 技術リスク
1. **リファクタリング中の機能停止**: 段階的移行により最小化
2. **既存機能の動作不良**: 包括的テストにより対応
3. **パフォーマンス低下**: ベンチマークテストで監視

### 運用リスク
1. **ユーザーへの影響**: 機能互換性を維持
2. **データ損失**: バックアップ体制強化
3. **開発期間延長**: バッファ期間を設定

## 成功指標

### 定量指標
- [ ] ビルド時間 < 30秒
- [ ] 単体テストカバレッジ > 80%
- [ ] 統合テスト通過率 > 95%
- [ ] Discord応答時間 < 2秒（平均）
- [ ] AI解析成功率 > 90%

### 定性指標
- [ ] コード可読性の向上
- [ ] 新機能追加の容易性
- [ ] エラー原因特定の迅速化
- [ ] 開発者体験の改善

## 次のアクション

1. **Phase 1の開始承認**: リファクタリング計画の最終確認
2. **バックアップ作成**: 現在のコードベースの完全バックアップ
3. **テスト環境構築**: リファクタリング用の独立環境
4. **モニタリング設定**: パフォーマンス測定ツールの導入

---

**作成日**: 2025-08-29  
**バージョン**: 1.0  
**ステータス**: 提案・レビュー待ち