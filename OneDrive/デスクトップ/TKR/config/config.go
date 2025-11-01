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
}

var (
	cfg Config
	// ▼▼▼【修正】RWMex -> RWMutex ▼▼▼
	mu sync.RWMutex
	// ▲▲▲【修正ここまで】▲▲▲
)

const configFilePath = "./tkr_config.json"

func LoadConfig() (Config, error) {
	mu.RLock()
	defer mu.RUnlock()

	file, err := os.ReadFile(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var tempCfg Config
	if err := json.Unmarshal(file, &tempCfg); err != nil {
		return Config{}, err
	}
	cfg = tempCfg
	return cfg, nil
}

func SaveConfig(newCfg Config) error {
	mu.Lock()
	defer mu.Unlock()

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
