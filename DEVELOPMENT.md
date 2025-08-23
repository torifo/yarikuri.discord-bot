# 開発者向けドキュメント - Yarikuri Discord Bot

このドキュメントは、個人の家計管理Discord Botの技術的詳細、ビルド方法、実装仕様、および使用方法について記載しています。

## セットアップ・実行方法

### 前提条件
- Go 1.18以上
- PostgreSQL 16.2（マスターデータ用）
- Discord Bot Token
- Linux環境（Ubuntu推奨）

### 初回セットアップ

1. **環境変数の設定**
```bash
# .envファイルを作成
cd /home/ubuntu/Bot/discord/yarikuri
echo "TOKEN=your_discord_bot_token_here" > .env
```

2. **依存関係のインストール**
```bash
cd bot
go mod download
```

3. **ビルド**
```bash
# 開発用ビルド
go build -o yarikuri_bot main.go

# 本番用ビルド（最適化）
go build -ldflags="-w -s" -o yarikuri_bot main.go

# 実行権限付与
chmod +x yarikuri_bot
```

4. **systemdサービス設定**
```bash
# サービスファイルの配置
sudo cp yarikuri_bot.service /etc/systemd/system/

# サービスの有効化
sudo systemctl daemon-reload
sudo systemctl enable yarikuri_bot
sudo systemctl start yarikuri_bot
sudo systemctl restart yarikuri_bot

# 状態確認
sudo systemctl status yarikuri_bot
```

### 実行方法

#### 開発時の実行
```bash
cd /home/ubuntu/Bot/discord/yarikuri/bot
go run main.go
```

#### 本番運用（systemd）
```bash
# 開始
sudo systemctl start yarikuri_bot

# 停止
sudo systemctl stop yarikuri_bot

# 再起動
sudo systemctl restart yarikuri_bot

# ログ確認
sudo journalctl -u yarikuri_bot -f
```

### テスト・デバッグ

```bash
# テスト実行（将来的にテストファイル追加予定）
go test ./...

# カバレッジ付きテスト
go test -cover ./...

# レースコンディション検出
go run -race main.go
```

## 使用方法

### Discord コマンド

#### `/check_master`
- **機能**: 各マスターデータの読み込み件数を表示
- **引数**: なし
- **用途**: データ読み込み状況の確認、デバッグ時の動作確認

**実行例**:
```
/check_master
```

**レスポンス例**:
```
マスターデータ読み込み状況
カテゴリ: 15件
グループ: 3件
ユーザー: 2件
支払い方法: 8件
収入源: 4件
収入種別: 3件
支払い種別: 6件
```

#### `/show_master`
- **機能**: 指定したマスターデータの詳細一覧を表示
- **引数**: `type` (category/group/user/payment_type)
- **制限**: Discord文字数制限(2000文字)により、大量データは一部省略

**実行例**:
```
/show_master type:category
/show_master type:payment_type
```

**レスポンス例**:
```
カテゴリ一覧
ID: 1, Name: 食費
ID: 2, Name: 交通費
ID: 3, Name: 光熱費
...
```

### 運用時のメンテナンス

#### ログ監視
```bash
# リアルタイムログ確認
sudo journalctl -u yarikuri_bot -f

# 特定期間のログ確認
sudo journalctl -u yarikuri_bot --since "2025-01-01" --until "2025-01-02"

# エラーログのみ抽出
sudo journalctl -u yarikuri_bot -p err
```

#### マスターデータ更新
マスターデータを更新する場合：
1. PostgreSQLでデータ更新
2. `pg_dump` でダンプファイル再生成
3. ボット再起動で新データ読み込み

```bash
# データ更新後の再起動
sudo systemctl restart yarikuri_bot
```

## アーキテクチャ概要

### データフロー

```
PostgreSQL Dump → File Reader → In-Memory Structures → Discord Commands
     ↓              ↓                    ↓                   ↓
master_data_dump → parseTableData() → Global Variables → Command Handlers
```

### 主要コンポーネント

#### 1. データ構造体
- [`Category`](bot/main.go:18): カテゴリ情報
- [`Group`](bot/main.go:19): グループ情報  
- [`PaymentType`](bot/main.go:20): 支払い方法
- [`User`](bot/main.go:21): ユーザー情報
- [`SourceList`](bot/main.go:22): 収入源
- [`TypeKind`](bot/main.go:23): 収入種別
- [`TypeList`](bot/main.go:24): 支払い種別

#### 2. データ処理関数
- [`parseTableData()`](bot/main.go:38-57): SQLダンプからテーブルデータを抽出
- [`loadMasterData()`](bot/main.go:59-117): マスターデータをメモリに読み込み

#### 3. Discord インタラクション
- [`commands`](bot/main.go:120-143): スラッシュコマンド定義
- [`commandHandlers`](bot/main.go:146-219): コマンド処理関数

## 実装済み機能詳細

### 1. マスターデータ管理

**機能概要**: PostgreSQL ダンプファイルから7つのマスターテーブルを読み込み、メモリ上で管理

**実装詳細**:
```go
// グローバル変数でマスターデータを保持
var (
    masterCategories   []Category      // カテゴリ一覧
    masterGroups       []Group         // グループ一覧  
    masterPaymentTypes []PaymentType   // 支払い方法
    masterUsers        []User          // ユーザー一覧
    masterSourceList   []SourceList    // 収入源
    masterTypeKind     []TypeKind      // 収入種別
    masterTypeList     []TypeList      // 支払い種別
)
```

**処理フロー**:
1. [`os.ReadFile()`](bot/main.go:62) でSQLダンプを読み込み
2. [`parseTableData()`](bot/main.go:38) でテーブル別にデータ抽出
3. 各構造体スライスに格納してメモリ保持

### 2. Discord スラッシュコマンド

#### `/check_master` コマンド
- **機能**: 各マスターデータの読み込み件数を表示
- **実装**: [`commandHandlers["check_master"]`](bot/main.go:147-165)
- **レスポンス**: Embed形式で件数を表示

#### `/show_master` コマンド  
- **機能**: 指定したマスターデータの詳細一覧を表示
- **パラメータ**: `type` (category/group/user/payment_type)
- **実装**: [`commandHandlers["show_master"]`](bot/main.go:167-218)
- **制限**: Discord文字数制限(2000文字)に対応

### 3. システム運用機能

**systemd 統合**:
- サービス名: `yarikuri_bot`
- 自動再起動: 10秒間隔
- ログ出力: journald 統合
- 実行ユーザー: root
- 作業ディレクトリ: `/home/ubuntu/Bot/discord/yarikuri/bot`

## 今後の実装予定機能

### Phase 1: 基本的な家計簿機能

#### 1.1 支出記録機能
```go
// 予定する構造体
type Expense struct {
    ID          int       `json:"id"`
    UserID      int       `json:"user_id"`      // user_list.id
    CategoryID  int       `json:"category_id"`  // category_list.id  
    Amount      int       `json:"amount"`       // 金額（円）
    PaymentID   int       `json:"payment_id"`   // payment_type.pay_id
    Description string    `json:"description"`  // 摘要
    Date        time.Time `json:"date"`         // 支出日
    CreatedAt   time.Time `json:"created_at"`   // 登録日時
}
```

**実装予定コマンド**:
- `/expense add <金額> <カテゴリ> <支払い方法> [摘要]`: 支出登録
- `/expense list [期間]`: 支出一覧表示
- `/expense delete <ID>`: 支出削除

#### 1.2 収入記録機能  
```go
type Income struct {
    ID         int       `json:"id"`
    UserID     int       `json:"user_id"`     // user_list.id
    SourceID   int       `json:"source_id"`   // source_list.id
    TypeID     int       `json:"type_id"`     // type_kind.id
    Amount     int       `json:"amount"`      // 金額（円）
    Date       time.Time `json:"date"`        // 収入日
    CreatedAt  time.Time `json:"created_at"`  // 登録日時
}
```

**実装予定コマンド**:
- `/income add <金額> <収入源> <種別>`: 収入登録
- `/income list [期間]`: 収入一覧表示

### Phase 2: 分析・レポート機能

#### 2.1 月次サマリー機能
```go
type MonthlySummary struct {
    Month        string `json:"month"`         // YYYY-MM
    UserID       int    `json:"user_id"`
    TotalIncome  int    `json:"total_income"`  // 総収入
    TotalExpense int    `json:"total_expense"` // 総支出
    Balance      int    `json:"balance"`       // 収支差
    Categories   []CategorySummary `json:"categories"`
}

type CategorySummary struct {
    CategoryName string `json:"category_name"`
    Amount       int    `json:"amount"`
    Percentage   float64 `json:"percentage"`
}
```

**実装予定コマンド**:
- `/report monthly [年月]`: 月次レポート表示
- `/report category <年月>`: カテゴリ別支出分析

#### 2.2 予算管理機能
```go
type Budget struct {
    ID         int    `json:"id"`
    UserID     int    `json:"user_id"`
    CategoryID int    `json:"category_id"`
    Month      string `json:"month"`     // YYYY-MM
    Amount     int    `json:"amount"`    // 予算額
    CreatedAt  time.Time `json:"created_at"`
}
```

**実装予定コマンド**:
- `/budget set <カテゴリ> <金額> [年月]`: 予算設定
- `/budget status [年月]`: 予算達成状況確認

### Phase 3: 高度な機能

#### 3.1 データ可視化
- Discord Embed でのグラフ表示
- 月次推移チャート
- カテゴリ別円グラフ

#### 3.2 通知機能
- 予算超過アラート
- 月末サマリー自動送信
- 定期的な支出入力リマインド

#### 3.3 データエクスポート
- CSV形式でのデータ出力
- 期間指定でのデータ取得
- 統計データの生成

## データベース設計（拡張予定）

### 新規追加予定テーブル

```sql
-- 支出記録テーブル
CREATE TABLE expenses (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES user_list(id),
    category_id INTEGER REFERENCES category_list(id),
    payment_id INTEGER REFERENCES payment_type(pay_id),
    amount INTEGER NOT NULL,
    description TEXT,
    expense_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 収入記録テーブル  
CREATE TABLE incomes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES user_list(id),
    source_id INTEGER REFERENCES source_list(id),
    type_id INTEGER REFERENCES type_kind(id),
    amount INTEGER NOT NULL,
    income_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 予算管理テーブル
CREATE TABLE budgets (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES user_list(id),
    category_id INTEGER REFERENCES category_list(id),
    month CHARACTER(7) NOT NULL, -- YYYY-MM
    amount INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, category_id, month)
);
```

## 開発ガイドライン

### コーディング規約
1. **命名規則**: キャメルケース（Go標準）
2. **コメント**: 日本語コメント推奨（ドメイン特化のため）
3. **エラーハンドリング**: 必ず適切なログ出力を行う
4. **構造体**: JSONタグを必須で付与
5. **個人用途**: 他ユーザーへの配慮は不要、自分の使いやすさを最優先

### Git運用
- **main**: 本番リリース用
- **fix-crontab**: 現在の開発ブランチ
- **feature/***: 機能別開発ブランチ

### テスト方針
1. **ユニットテスト**: 各関数の単体テスト
2. **インテグレーションテスト**: Discord API との統合テスト
3. **E2Eテスト**: 実際のコマンド実行テスト
4. **個人検証**: 自分の使用環境での動作確認を重視

### 開発環境構築

```bash
# Go の開発ツールインストール
go install golang.org/x/tools/gopls@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/cosmtrek/air@latest  # ホットリロード用

# VSCode拡張機能推奨
# - Go (Google)
# - Go Test Explorer
# - Thunder Client (API テスト用)
```

### 個人開発の利点
- **迅速な意思決定**: 要件変更・仕様変更を即座に実装可能
- **完全カスタマイズ**: 自分の家計管理スタイルに100%合わせた設計
- **学習効果**: 全体アーキテクチャを把握した状態での継続開発
- **データ所有**: 家計データの完全な管理権限

## トラブルシューティング

### よくある問題と解決方法

1. **マスターデータ読み込みエラー**
   - SQLダンプファイルのパス確認: `/home/ubuntu/Bot/discord/yarikuri/dump_local_db/master_data_dump.sql`
   - ファイル権限確認: `chmod 644 master_data_dump.sql`

2. **Discord接続エラー**  
   - TOKEN環境変数の設定確認
   - Bot権限（applications.commands）の確認

3. **systemdサービス起動失敗**
   - 実行ファイルのパス確認: `/home/ubuntu/Bot/discord/yarikuri/bot/yarikuri_bot`
   - 実行権限の確認: `chmod +x yarikuri_bot`
   - 作業ディレクトリの確認: `/home/ubuntu/Bot/discord/yarikuri/bot`
   - 環境変数の設定確認: `.env`ファイルの存在とTOKEN設定

4. **Discord Bot権限エラー**
   - Bot招待時の権限確認: `applications.commands`スコープが必要
   - サーバー権限: スラッシュコマンド使用権限の確認

5. **メモリ不足（大量データ時）**
   - マスターデータサイズの確認
   - システムメモリ使用量の監視: `free -h`
   - 必要に応じてデータの分割読み込み実装を検討