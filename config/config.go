package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	UsageFolderPath       string `json:"usageFolderPath"`
	DatFolderPath         string `json:"datFolderPath"`
	CalculationPeriodDays int    `json:"calculationPeriodDays"`
	MedicodeUserID        string `json:"medicodeUserID"`
	MedicodePassword      string `json:"medicodePassword"`
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

	if cfg.CalculationPeriodDays == 0 {
		cfg.CalculationPeriodDays = 90
	}

	return cfg, nil
}

func SaveConfig(newCfg Config) error {
	mu.Lock()
	defer mu.Unlock()

	if newCfg.CalculationPeriodDays == 0 {
		newCfg.CalculationPeriodDays = 90
	}

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
