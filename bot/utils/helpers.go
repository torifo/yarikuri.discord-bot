package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/yarikuri/errors"
)

// DownloadImage は指定されたURLから画像をダウンロードして一時ファイルに保存
func DownloadImage(url, tempDir string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", errors.NewBotError(errors.ErrorTypeNetwork, "画像URLへのHTTPリクエストに失敗", err).
			WithContext("url", url)
	}
	defer response.Body.Close()

	// ディレクトリが存在しない場合は作成
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", errors.NewBotError(errors.ErrorTypeFileIO, "一時ディレクトリの作成に失敗", err).
			WithContext("temp_dir", tempDir)
	}

	filePath := filepath.Join(tempDir, filepath.Base(response.Request.URL.Path))
	file, err := os.Create(filePath)
	if err != nil {
		return "", errors.NewBotError(errors.ErrorTypeFileIO, "一時画像ファイルの作成に失敗", err).
			WithContext("file_path", filePath)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", errors.NewBotError(errors.ErrorTypeFileIO, "画像データの書き込みに失敗", err).
			WithContext("file_path", filePath)
	}

	return filePath, nil
}

// EnsureDir はディレクトリが存在しない場合に作成する
func EnsureDir(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return errors.NewBotError(errors.ErrorTypeFileIO, "ディレクトリの作成に失敗", err).
			WithContext("dir_path", dirPath)
	}
	return nil
}