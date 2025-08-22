# yarikuri_bot 開発トラブルシューティング

このドキュメントは、yarikuri_botの開発過程で発生した問題とその解決策をまとめたものです。

## 1. ローカル環境の問題

### 1.1. psqlコマンドの文字化け

**問題**: WindowsのPowerShellでpsqlを使い`\d`コマンドを実行すると、テーブル名などの日本語が文字化けする。

**原因**: PowerShellのデフォルト文字コード（Shift_JIS）と、psqlが使用するUTF-8が一致しないため。

**解決策**:
- psql実行前に`chcp 65001`コマンドでターミナルのコードページをUTF-8に変更する。
- ターミナルのプロパティから、フォントをMS Gothicなどの日本語対応フォントに変更する。
- **推奨**: より文字コードの扱いに強いWindows Terminalを導入する。

### 1.2. シェルスクリプトが実行できない

**問題**: PowerShellで`chmod +x`や`./script.sh`がエラーになる。

**原因**: `chmod`はLinux/macOS用のコマンドであり、PowerShellでは使用できない。また、PowerShellはデフォルトで`.sh`ファイルの実行を許可していない。

**解決策**: Windows上でLinuxコマンド環境を再現できるGit Bashを使い、スクリプトを実行する。

## 2. データベースとデータの問題

### 2.1. PostgreSQLへの接続失敗

**問題**: Connection refusedエラーが発生し、DBに接続できない。

**原因**: PostgreSQLのサーバープロセス自体が起動していなかった。

**解決策**: Windowsのサービス管理ツール(`services.msc`)からpostgresqlサービスを開始する。

### 2.2. pg_dumpとSQLiteの非互換性

**問題**: `pg_dump`で作成したSQLファイルを`sqlite3`で読み込むと、大量の構文エラーが発生する。

**原因**: `pg_dump`が生成するSQLには、`SET`文や`OWNER TO`など、SQLiteが解釈できないPostgreSQL固有の構文が含まれているため。

**解決策**: `pg_dump`で生成したファイルを直接SQLiteに読み込ませるのをやめ、Goプログラム側で`COPY`文を解釈し、必要なデータだけをメモリに格納する方式に変更した。

## 3. Go Bot開発の問題

### 3.1. ビルドエラー cannot find main module

**問題**: `go build`コマンドが失敗する。

**原因**: プロジェクトのルートディレクトリに、Goのプロジェクト定義ファイルである`go.mod`が存在しなかった。

**解決策**: Botのソースコードがあるディレクトリで`go mod init yarikuri`を実行し、プロジェクトを正しく初期化する。

### 3.2. サービスが起動直後にクラッシュする

**問題**: `systemctl status`で確認すると、`Active: activating (auto-restart)`と`status=1/FAILURE`を繰り返している。

**原因**: systemdサービスが`.env`ファイルを読み込めておらず、Botが必須のTOKEN環境変数を取得できずにパニックを起こしていた。

**解決策**: `.service`ファイルの`[Service]`セクションに`EnvironmentFile=/path/to/bot/.env`という行を追加し、systemdに`.env`ファイルの場所を明示的に教える。

## 4. Discord APIとBotの挙動の問題

### 4.1. 画像メッセージにBotが反応しない

**問題**: 画像を投稿しても、messageCreateイベントハンドラが動作しない。

**原因**: メッセージの内容（添付ファイルを含む）を読み取るために必要なMessage Content IntentがDiscord Developer Portalで有効になっていなかった。

**解決策**: Developer PortalのBot設定ページで、「Privileged Gateway Intents」セクションにある「MESSAGE CONTENT INTENT」をONにする。

### 4.2. 「インタラクションに失敗しました」エラー

**問題**: `/show_master`コマンドでページ送りボタンを押すと、エラーが表示される。

**原因**: Botがボタン操作に対して3秒以内に応答できなかったため。ページをめくるたびにデータのソートやマップ作成を行っており、処理が遅延していた。

**解決策**:
- **処理の高速化**: データのソートや検索用マップの作成を、Bot起動時に一度だけ行うように変更。
- **遅延応答の実装**: ボタンが押されたら、まず`InteractionResponseDeferredMessageUpdate`で「処理中」であることをDiscordに伝え、3秒のタイムアウトを回避。その後、ゆっくりとメッセージ内容を生成し、`InteractionResponseEdit`で最終的な応答を返す。

### 4.3. 特定の項目でページ遷移ができない

**問題**: `/show_master`の「支払い方法」でのみ、ページ遷移が機能しない。

**原因**: ボタンのCustomIDを`paginate_payment_type_1`のように`_`で区切っていたが、データタイプ名`payment_type`自体に`_`が含まれていたため、IDの分解に失敗していた。


### 3.3. Google Generative AI Go SDKのインストールエラー

**問題**: `go get github.com/google/generative-ai-go/genai`および`go get google.golang.org/api/option`実行時に以下のエラーが発生する：
```
cmp: package cmp is not in GOROOT (/usr/lib/go-1.18/src/cmp)
slices: package slices is not in GOROOT (/usr/lib/go-1.18/src/slices)
log/slog: package log/slog is not in GOROOT (/usr/lib/go-1.18/src/log/slog)
math/rand/v2: package math/rand/v2 is not in GOROOT (/usr/lib/go-1.18/src/math/rand/v2)
maps: package maps is not in GOROOT (/usr/lib/go-1.18/src/maps)
```

**原因**: 使用していたGo 1.18では、Google Generative AI Go SDKが必要とする新しい標準ライブラリパッケージ（`cmp`, `slices`, `log/slog`, `maps`, `math/rand/v2`）が含まれていない。これらのパッケージはGo 1.21以降で導入された。

**解決策**:
1. **Goのバージョンアップグレード**:
   ```bash
   # 最新のGoをダウンロード
   wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
   
   # 古いGoを削除し、新しいGoをインストール
   sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
   
   # PATHの設定
   export PATH=/usr/local/go/bin:$PATH
   
   # バージョン確認
   go version  # go version go1.23.4 linux/amd64
   ```

2. **go.modファイルの更新**:
   ```go
   // go.modファイル内のGoバージョンを更新
   go 1.23  // 1.18から変更
   ```

3. **ライブラリの再インストール**:
   ```bash
   cd bot
   go clean -modcache
   go get github.com/google/generative-ai-go/genai
   go get google.golang.org/api/option
   go mod tidy
   ```

4. **ビルドの確認**:
   ```bash
   go build -ldflags="-w -s" -o yarikuri_bot main.go
   ```

**注意**: Goのアップグレード後は、未使用のimportや未定義関数などの既存のコードエラーも修正する必要がある場合があります。
**解決策**: CustomIDの区切り文字を、データタイプ名には含まれない**`:`（コロン）**に変更した。（例: `paginate:payment_type:1`）