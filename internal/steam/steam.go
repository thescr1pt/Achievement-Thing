package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var cacheDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "Achievement-Thing", "cache")

var recentCacheOperations = make(map[string]time.Time)
var recentCacheOperationsMutex sync.RWMutex

const cacheCooldown = 5 * time.Second

func isOlderThanMonths(filepath string, months int) (bool, error) {
	info, err := os.Stat(filepath)
	if err != nil {
		return false, err
	}
	modTime := info.ModTime()
	return time.Since(modTime) > time.Duration(months)*30*24*time.Hour, nil
}

type Achievement struct {
	Name         string `json:"name"`
	DisplayName  string `json:"displayName"`
	Description  string `json:"description,omitempty"`
	Icon         string `json:"icon"`
	IconGray     string `json:"icongray"`
	Hidden       int    `json:"hidden"`
	DefaultValue int    `json:"defaultvalue"`
}

type AchievementsData struct {
	AppID        string        `json:"appid"`
	Name         string        `json:"name"`
	Achievements []Achievement `json:"achievements"`
}

func CacheAchievements(apikey string, appid string) error {
	fmt.Println("Caching achievements for appId:", appid)
	if apikey == "" || appid == "" {
		return errors.New("API key or App ID is empty")
	}

	recentCacheOperationsMutex.Lock()
	if lastOp, exists := recentCacheOperations[appid]; exists {
		if time.Since(lastOp) < cacheCooldown {
			recentCacheOperationsMutex.Unlock()
			return nil
		}
	}
	recentCacheOperations[appid] = time.Now()
	recentCacheOperationsMutex.Unlock()

	cacheFilePath := filepath.Join(cacheDir, appid, "achievements.json")
	// Check if cache file exists and is recent
	if _, err := os.Stat(cacheFilePath); err == nil {
		isOld, err := isOlderThanMonths(cacheFilePath, 3)
		if err != nil {
			fmt.Println("Error checking cache file age:", err)
			return err
		}
		if !isOld {
			fmt.Println("Cache file is recent, skipping fetch for appId:", appid)
			return nil
		}

	}

	url := "https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=" + apikey + "&appid=" + appid
	// fmt.Println("Fetching achievements from URL:", url)

	if err := os.MkdirAll(filepath.Dir(cacheFilePath), 0755); err != nil {
		fmt.Println("Error creating cache directory:", err)
		return err
	}

	//get the data from the url and save it to cacheFilePath
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching data from API:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Non-OK HTTP status:", resp.StatusCode)
		return errors.New("failed to fetch data from API")
	}

	var apiResponse struct {
		Game struct {
			GameName           string `json:"gameName"`
			AvailableGameStats struct {
				Achievements []Achievement `json:"achievements"`
			} `json:"availableGameStats"`
		} `json:"game"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		fmt.Println("Error decoding API response:", err)
		return err
	}

	achievementsData := AchievementsData{
		AppID:        appid,
		Name:         apiResponse.Game.GameName,
		Achievements: apiResponse.Game.AvailableGameStats.Achievements,
	}

	file, err := os.Create(cacheFilePath)
	if err != nil {
		fmt.Println("Error creating cache file:", err)
		return err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(achievementsData); err != nil {
		fmt.Println("Error writing to cache file:", err)
		return err
	}

	return nil
}

func GetAchievement(appid string, achievementName string, apikey string) (*Achievement, error) {
	cacheFilePath := filepath.Join(cacheDir, appid, "achievements.json")
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		if err := CacheAchievements(apikey, appid); err != nil {
			return nil, err
		}
	}

	file, err := os.Open(cacheFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var achievementsData AchievementsData
	if err := json.NewDecoder(file).Decode(&achievementsData); err != nil {
		return nil, err
	}

	for _, achievement := range achievementsData.Achievements {
		if achievement.Name == achievementName {
			return &achievement, nil
		}
	}

	return nil, errors.New("achievement not found")
}

func GetImage(appid string, imageURL string) (string, error) {
	imageDir := filepath.Join(cacheDir, appid, "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", err
	}
	imagePath := filepath.Join(imageDir, filepath.Base(imageURL))

	if _, err := os.Stat(imagePath); err == nil {
		isOld, err := isOlderThanMonths(imagePath, 6)
		if err != nil {
			return "", err
		}
		if !isOld {
			return imagePath, nil
		}
	}

	// Download the image and save it to the cache
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to fetch image")
	}

	file, err := os.Create(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}

	return imagePath, nil
}
