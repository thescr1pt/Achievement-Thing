package settingservice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Settings struct {
	ApiKey  string   `json:"apiKey"`
	Folders []string `json:"folders"`
}

// settingsPath = %localappdata%\Achievement-Thing\settings.json
var settingsPath = filepath.Join(os.Getenv("LOCALAPPDATA"), "Achievement-Thing", "settings.json")

func getDefaultFolders() []string {
	return []string{
		filepath.Join(os.Getenv("PUBLIC"), "Documents", "Steam", "CODEX"),
		filepath.Join(os.Getenv("PUBLIC"), "Documents", "Steam", "RUNE"),
		filepath.Join(os.Getenv("PUBLIC"), "Documents", "OnlineFix"),
		filepath.Join(os.Getenv("PUBLIC"), "Documents", "Empress"),
		filepath.Join(os.Getenv("APPDATA"), "Empress"),
		filepath.Join(os.Getenv("APPDATA"), "Steam", "CODEX"),
		filepath.Join(os.Getenv("APPDATA"), "SmartSteamEmu"),
		filepath.Join(os.Getenv("APPDATA"), "CreamAPI"),
		filepath.Join(os.Getenv("PROGRAMDATA"), "Steam"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "skidrow"),
	}
}

func createDefaultSettings() Settings {
	var defaultSettings = Settings{
		ApiKey:  "",
		Folders: getDefaultFolders(),
	}
	return defaultSettings
}

func saveSettings(settings Settings) error {
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating settings directory: %w", err)
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("error marshalling settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, settingsJSON, 0644); err != nil {
		return fmt.Errorf("error writing settings file: %w", err)
	}
	return nil
}

func LoadSettings() (Settings, error) {
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		defaultSettings := createDefaultSettings()
		if err := saveSettings(defaultSettings); err != nil {
			return Settings{}, err
		}
		return defaultSettings, nil
	}
	settingsFile, err := os.ReadFile(settingsPath)
	if err != nil {
		return Settings{}, fmt.Errorf("error reading settings file: %w", err)
	}
	var loadedSettings Settings
	if err := json.Unmarshal(settingsFile, &loadedSettings); err != nil {
		return Settings{}, fmt.Errorf("error unmarshalling settings: %w", err)
	}
	return loadedSettings, nil
}

func GetPath() string {
	return settingsPath
}
