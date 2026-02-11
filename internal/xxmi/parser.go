package xxmi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Importers map[string]struct {
		Importer struct {
			GameFolder   string   `json:"game_folder"`
			GameExeNames []string `json:"game_exe_names"`
		} `json:"Importer"`
	} `json:"Importers"`
}

func GetGameFilepath(loaderName string) (string, string, error) {
	configPath, err := filepath.Abs(filepath.Join("..", "XXMI Launcher Config.json"))
	if err != nil {
        return "", "", fmt.Errorf("could not find config file: %w", err)
    }

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal config: %w", err)
	}

	wrapper, exists := cfg.Importers[loaderName]
	if !exists {
		return "", "", fmt.Errorf("loader '%s' not found in config", loaderName)
	}

	folder := wrapper.Importer.GameFolder
	if folder == "" {
		return "", "", fmt.Errorf("game folder for '%s' is empty in config", loaderName)
	}

	if len(wrapper.Importer.GameExeNames) == 0 {
		return folder, "", fmt.Errorf("no executable names found for '%s'", loaderName)
	}

	return folder, wrapper.Importer.GameExeNames[0], nil
}

func GetLauncherFilepath() (string, error) {
    launcherPath, err := filepath.Abs(filepath.Join("..", "Resources", "Bin", "XXMI Launcher.exe"))
    if err != nil {
        return "", err
    }

    return launcherPath, nil
}