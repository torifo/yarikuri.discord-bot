package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yarikuri/agent"
	"github.com/yarikuri/config"
	"github.com/yarikuri/data"
	"github.com/yarikuri/discord"
	"github.com/yarikuri/errors"
	"github.com/yarikuri/models"
)

func main() {
	log.Println("Yarikuri Discord Bot を起動中...")

	// 設定を読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		errors.HandleError(err, nil)
		log.Fatal(err)
	}

	constants := config.DefaultConstants()

	// Bot状態を初期化
	state := models.NewBotState()
	state.TargetChannelID = cfg.TargetChannelID

	// マスターデータを読み込み
	err = data.LoadMasterData("master.sql", state)
	if err != nil {
		errors.HandleError(err, nil)
		log.Fatal(err)
	}

	// AIクライアントを初期化
	aiClient, err := agent.NewClient(cfg.GeminiAPIKey)
	if err != nil {
		errors.HandleError(err, nil)
		log.Fatal(err)
	}
	state.GeminiClient = aiClient.GetModel()

	// キューマネージャを初期化
	queueManager := data.NewQueueManager(constants.QueueFilePath)
	err = queueManager.LoadFromFile()
	if err != nil {
		errors.HandleError(err, nil)
		log.Printf("キューファイルの読み込みに失敗: %v", err)
	}

	// Discordセッションを開始
	sessionManager, err := discord.NewSessionManager(cfg.DiscordToken, state)
	if err != nil {
		errors.HandleError(err, nil)
		log.Fatal(err)
	}

	err = sessionManager.Start()
	if err != nil {
		errors.HandleError(err, nil)
		log.Fatal(err)
	}

	log.Println("Bot が正常に起動しました。終了するには Ctrl+C を押してください。")

	// 終了シグナルを待機
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("シャットダウン中...")

	// セッションを終了
	err = sessionManager.Close()
	if err != nil {
		log.Printf("セッション終了エラー: %v", err)
	}

	log.Println("Bot が終了しました。")
}