// C:\Users\wasab\OneDrive\デスクトップ\TKR\config_handler.go
package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"tkr/config"
)

// GetConfigHandler は現在の設定を返します
func GetConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := config.GetConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	}
}

// ▼▼▼【修正】SaveConfigHandler で DatFolderPath も検証・保存する ▼▼▼
func SaveConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			http.Error(w, "リクエストが不正です。", http.StatusBadRequest)
			return
		}

		// フォルダパスの検証 (処方取込パス)
		if err := validateFolderPath(newCfg.UsageFolderPath); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// フォルダパスの検証 (伝票取込パス)
		if err := validateFolderPath(newCfg.DatFolderPath); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := config.SaveConfig(newCfg); err != nil {
			log.Printf("Error saving config: %v", err)
			http.Error(w, "設定の保存に失敗しました。", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "設定を保存しました。"})
	}
}

// フォルダパスを検証するヘルパー関数
func validateFolderPath(path string) error {
	if path == "" {
		return nil // 空の場合は検証しない
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("指定されたフォルダパスが見つかりません: " + path)
		}
		log.Printf("Error checking folder path: %v", err)
		return errors.New("フォルダパスの確認中にエラーが発生しました。")
	}
	if !info.IsDir() {
		return errors.New("指定されたパスはフォルダではありません: " + path)
	}
	return nil
}

// ▲▲▲【修正ここまで】▲▲▲
