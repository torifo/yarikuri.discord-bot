# Yarikuri Discord Bot - 統合リファクタリング計画書

## 概要

本文書は、家計管理Discord Bot「yarikuri_bot」の現在のコードベース分析に基づく、**実用的なパッケージ分割戦略**を提案します。現在の[`main.go`](../bot/main.go)は3344行となっており、保守性・拡張性を向上させながら、適切なレベルでの機能分離を目指します。

## 現状分析

### ファイルサイズと構造
- **総行数**: 3344行（モノリシック構造）
- **主要関数数**: 50個以上のハンドラ関数
- **構造体数**: 11個の主要データ構造
- **グローバル変数**: 15個以上の状態管理変数

### 機能分析
1. **エラーハンドリング統一** (30-108行) - `BotError`構造体と統一エラーハンドリング
2. **グローバル変数定義** (112-133行) - Discord、AI、マスターデータのグローバル変数
3. **構造体定義** (143-204行) - データ構造の定義
4. **データ読み込み・解析関連** (209-332行) - マスターデータ読み込み・パース
5. **Discordコマンド定義** (337-380行) - コマンド登録とハンドラマップ
6. **Discordイベントハンドラ** (386-3344行) - 大量のイベントハンドラ関数群

### 主要課題
1. **単一責任原則の違反**: 一つのファイルが複数の責任を持つ
2. **高い結合度**: 機能間の依存関係が複雑
3. **テストの困難性**: モノリシック構造によるユニットテスト実装困難
4. **保守性の低下**: コード変更時の影響範囲の特定が困難

## 新しいパッケージ構造（バランス版）

前回の詳細構造と簡潔構造の中間を取った実用的な構造：

```
bot/
├── main.go                    # エントリポイント（最小限）
├── config/
│   └── config.go              # 設定管理・定数定義
├── models/
│   ├── structures.go          # データ構造定義
│   └── state.go               # 状態管理構造体
├── errors/
│   └── bot_error.go           # 統一エラーハンドリング
├── data/
│   ├── master.go              # マスターデータ管理
│   ├── queue.go               # キューファイル管理
│   └── parser.go              # データパース機能
├── discord/
│   ├── commands.go            # コマンド定義
│   ├── session.go             # セッション管理
│   ├── handlers/
│   │   ├── message.go         # メッセージハンドラ
│   │   ├── buttons.go         # ボタンハンドラ
│   │   ├── modals.go          # モーダルハンドラ
│   │   └── selects.go         # セレクトメニューハンドラ
│   └── ui/
│       ├── components.go      # UI コンポーネント
│       └── embeds.go          # Embed メッセージ
├── agent/
│   ├── client.go              # Gemini クライアント
│   ├── analyzer.go            # レシート解析
│   └── prompts.go             # プロンプト管理
├── payment/
│   ├── classifier.go          # 支払い方法分類
│   ├── details.go             # 支払い詳細管理
│   └── splitter.go            # 金額分割機能
└── utils/
    ├── sorting.go             # ソート機能
    └── helpers.go             # ヘルパー関数
```

### パッケージ責任分担

#### **main.go** (~100行)
- エントリポイント
- 初期化処理のオーケストレーション
- シャットダウン処理

#### **config/** (~150行)
- `config.go`: 環境変数管理、設定構造体、定数定義、設定値

#### **models/** (~250行)
- `structures.go`: 全データ構造定義
- `state.go`: トランザクション・確認データ状態管理

#### **errors/** (~100行)
- `bot_error.go`: BotError構造体、統一エラーハンドリング、ログ機能

#### **data/** (~500行)
- `master.go`: マスターデータ読み込み・管理
- `queue.go`: キューファイル管理機能
- `parser.go`: SQLダンプパース機能

#### **discord/** (~300行 + handlers/)
- `commands.go`: コマンド定義・登録
- `session.go`: Discord セッション管理
- `handlers/` ディレクトリ:
  - `message.go`: メッセージハンドラ (~200行)
  - `buttons.go`: ボタンハンドラ群 (~400行)
  - `modals.go`: モーダルハンドラ群 (~300行)
  - `selects.go`: セレクトメニューハンドラ群 (~400行)
- `ui/` ディレクトリ:
  - `components.go`: UI コンポーネント生成 (~200行)
  - `embeds.go`: Embed メッセージ作成 (~100行)

#### **agent/** (~300行)
- `client.go`: Gemini クライアント設定
- `analyzer.go`: レシート解析・バックグラウンド処理
- `prompts.go`: AIプロンプト管理

#### **payment/** (~400行)
- `classifier.go`: 支払い方法分類ロジック
- `details.go`: 支払い詳細管理・マッチング機能
- `splitter.go`: 金額分割機能

#### **utils/** (~150行)
- `sorting.go`: 日本語ソート機能
- `helpers.go`: ヘルパー関数、ファイル操作

## 実装戦略

### Phase 1: 基盤パッケージの作成 (Week 1)
1. **models/**: データ構造定義の移行
2. **config/**: 設定管理の整備
3. **errors/**: エラーハンドリング統一
4. **utils/**: 共通ユーティリティの分離

### Phase 2: データ・AI機能分離 (Week 2)
1. **data/**: マスターデータ管理機能の移行
2. **agent/**: AI解析機能の分離
3. **payment/**: 支払い関連ロジックの分離

### Phase 3: Discord機能分離 (Week 3-4)
1. **discord/commands.go**: コマンド定義の分離
2. **discord/handlers/**: 大量のハンドラ関数群の機能別分離
3. **discord/ui/**: UI コンポーネントの分離

### Phase 4: 最終調整 (Week 5)
1. **main.go の最小化**: エントリポイントのみに絞る
2. **依存関係の整理**: パッケージ間の依存を明確化
3. **全機能の動作確認**: 既存機能の完全動作保証

## グローバル状態の管理

### 設計方針
```go
// config/config.go
type BotConfig struct {
    DiscordToken    string
    TargetChannelID string
    GeminiAPIKey    string
    // ... その他設定
}

type BotState struct {
    mu sync.RWMutex
    
    // Discord関連
    TargetChannelID string
    
    // AI関連  
    GeminiClient *genai.GenerativeModel
    
    // 状態管理マップ
    Transactions     map[string]*TransactionState
    ConfirmationData map[string]*ConfirmationData
    
    // マスターデータ（読み取り専用）
    MasterCategories   []Category
    MasterGroups       []Group
    MasterPaymentTypes []PaymentType
    // ... その他マスターデータ
    
    // キューデータ
    MasterDataQueues map[string][]MasterQueueItem
    QueueMutex      sync.RWMutex
}

var (
    Config *BotConfig
    State  *BotState
)
```

### アクセス制御
- 各パッケージは必要な状態のみアクセス
- 書き込みは適切なミューテックスで保護
- 読み取り専用データは効率的なアクセス提供

## 重要な考慮事項

### 既存機能の完全保持
1. **支払い方法分類ロジック**: `payment/classifier.go`で完全移行
2. **金額分割機能**: `payment/splitter.go`で保持
3. **AI解析結果処理**: `agent/analyzer.go`で継続
4. **全Discordコマンド**: `discord/`で動作維持

### パフォーマンス維持
- グローバル状態への効率的なアクセス
- パッケージ間通信のオーバーヘッド最小化
- 並行処理の適切な管理

## 期待される効果

### 定量的改善
1. **ファイルサイズ**: main.go を100行以下に削減
2. **機能維持**: 既存の全機能が正常動作
3. **保守性**: 機能変更時の影響範囲を80%削減
4. **可読性**: コードレビュー時間を50%短縮

### 定性的改善
1. **責任分離**: 各パッケージの責任が明確
2. **テスタビリティ**: パッケージ単位でのテスト実装
3. **拡張性**: 新機能追加の容易性
4. **保守性**: バグ修正時の影響範囲限定

## リスク管理

### 潜在的リスク
1. **機能破綻**: リファクタリング過程での機能損失
2. **循環依存**: パッケージ間の不適切な依存関係
3. **パフォーマンス低下**: パッケージ分割によるオーバーヘッド

### 対策
1. **段階的実装**: 小さな単位での変更と検証
2. **テスト駆動**: 各段階での動作確認
3. **バックアップ**: 各段階でのコミットとブランチ管理
4. **依存関係管理**: 循環依存の早期発見と解決

## 成功指標

### 定量指標
1. **コードサイズ**: main.go を100行以下に削減
2. **機能完全性**: 既存機能の100%動作保証
3. **パフォーマンス**: レスポンス時間の悪化なし
4. **保守性**: 新機能追加時間の50%短縮

### 定性指標
1. **可読性**: 機能の所在が明確
2. **拡張性**: 新機能追加が容易
3. **テスタビリティ**: ユニットテスト実装可能
4. **チーム開発**: 複数人での並行開発が容易

---
*ブランチ: refactor/modularize-main*  
*統合文書: REFACTORING_PROPOSAL.md との統合・改訂版*