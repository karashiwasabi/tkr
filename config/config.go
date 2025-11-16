// C:\Users\wasab\OneDrive\デスクトップ\TKR\config\config.go
package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	UsageFolderPath string `json:"usageFolderPath"`
	DatFolderPath   string `json:"datFolderPath"`
	// ▼▼▼【ここに追加】(WASABI: config.go より) ▼▼▼
	CalculationPeriodDays int `json:"calculationPeriodDays"`
	// ▲▲▲【追加ここまで】▲▲▲
}

var (
	cfg Config
	mu  sync.RWMutex
)

const configFilePath = "./tkr_config.json"

func LoadConfig() (Config, error) {
	mu.RLock()
	defer mu.RUnlock()

	file, err := os.ReadFile(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// ▼▼▼【修正】デフォルト値を設定 (WASABI: config.go より) ▼▼▼
			return Config{
				CalculationPeriodDays: 90,
			}, nil
		}
		return Config{}, err
	}

	var tempCfg Config
	if err := json.Unmarshal(file, &tempCfg); err != nil {
		return Config{}, err
	}
	cfg = tempCfg

	// ▼▼▼【ここに追加】ロード時に0ならデフォルト値を設定 ▼▼▼
	if cfg.CalculationPeriodDays == 0 {
		cfg.CalculationPeriodDays = 90
	}
	// ▲▲▲【追加ここまで】▲▲▲

	return cfg, nil
}

func SaveConfig(newCfg Config) error {
	mu.Lock()
	defer mu.Unlock()

	// ▼▼▼【ここに追加】保存時にも0ならデフォルト値を設定 ▼▼▼
	if newCfg.CalculationPeriodDays == 0 {
		newCfg.CalculationPeriodDays = 90
	}
	// ▲▲▲【追加ここまで】▲▲▲

	file, err := json.MarshalIndent(newCfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configFilePath, file, 0644); err != nil {
		return err
	}
	cfg = newCfg
	return nil
}

func GetConfig() Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}
