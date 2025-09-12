package watcherservice

import (
	"Achievement-Thing/internal/helper"
	"Achievement-Thing/internal/notifier"
	"Achievement-Thing/internal/parser"
	"Achievement-Thing/internal/settingservice"
	"Achievement-Thing/internal/steam"
	"Achievement-Thing/pkg/filewatcher"
	"fmt"
	"os"
)

var achievementFiles = []string{
	"achievements.ini",
	"achievements.json",
	"achiev.ini",
	"Achievements.ini",
}

var watcher *filewatcher.FileWatcher
var stopChan chan any
var currentAchievements = make(map[string]map[string]parser.Achievement)

var folders []string
var apiKey string

const maxNotifyAchievements = 2

func FileEventHandler(event filewatcher.EventType, path string) {
	fmt.Println("File event:", event, path)
	if path == "" {
		return
	}
	if apiKey == "" {
		fmt.Println("No API Key set, cannot fetch achievement info")
		return
	}
	appId := helper.ExtractAppId(path)
	if appId == "" {
		fmt.Println("Could not extract appId from path:", path)
		return
	}

	if event == filewatcher.FileCreated {
		err := steam.CacheAchievements(apiKey, appId)
		if err != nil {
			fmt.Println("Error caching achievements:", err)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer f.Close()
	achievements, err := parser.ParseFile(f, path)
	if err != nil {
		fmt.Println("Error parsing file:", err)
		return
	}

	oldAchievements, exists := currentAchievements[appId]
	if !exists {
		oldAchievements = make(map[string]parser.Achievement)
	}
	newAchievements := make([]parser.Achievement, 0)
	for k, v := range achievements {
		oldAch, ok := oldAchievements[k]
		if !ok || (ok && !oldAch.Achieved && v.Achieved) {
			newAchievements = append(newAchievements, v)
		}
	}
	if count := len(newAchievements); count > 0 && count <= maxNotifyAchievements {
		currentAchievements[appId] = achievements
		fmt.Println("New achievements for appId:", appId)
		for _, v := range newAchievements {
			fmt.Println("  New Achievement: ", v.Name)
			achievementInfo, err := steam.GetAchievement(appId, v.Name, apiKey)
			if err == nil {
				icon := ""
				if achievementInfo.Icon != "" {
					iconPath, err := steam.GetImage(appId, achievementInfo.Icon)
					if err == nil {
						icon = iconPath
					} else {
						fmt.Println("Error fetching achievement icon:", err)
					}
				}
				notifier.SendAchievement(achievementInfo.DisplayName, achievementInfo.Description, icon)
			} else {
				fmt.Println("Error fetching achievement info:", err)
			}
		}
	}
}

func initializeWatcher() error {
	settings, err := settingservice.LoadSettings()
	if err != nil {
		fmt.Println("Error loading settings:", err)
		return err
	}

	folders = settings.Folders
	apiKey = settings.ApiKey

	for _, folder := range folders {
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			fmt.Println("Folder does not exist, skipping:", folder)
			continue
		}
		files, err := helper.FindFilesRecursive(folder, achievementFiles)
		if err != nil {
			fmt.Println("Error finding files:", err)
			return err
		}
		for _, file := range files {
			appId := helper.ExtractAppId(file)
			if appId != "" {
				f, openErr := os.Open(file)
				if openErr != nil {
					fmt.Println("Error opening file:", openErr)
					continue
				}
				defer f.Close()
				achievements, err := parser.ParseFile(f, file)
				if err == nil {
					currentAchievements[appId] = achievements

					if len(achievements) > 0 {
						fmt.Println("Loaded achievements for appId:", appId)
						for k := range achievements {
							fmt.Println("  [", k, "]")
						}
					}
				} else {
					fmt.Println("Error parsing file:", err)
				}
				if apiKey != "" {
					go steam.CacheAchievements(apiKey, appId)
				}
			}
		}
	}
	return nil
}

func StartWatcher() error {
	fmt.Println("Starting file watcher...")
	if err := initializeWatcher(); err != nil {
		fmt.Println("Error initializing watcher:", err)
		return err
	}
	var err error
	watcher, err = filewatcher.New()
	if err != nil {
		fmt.Println("Error creating file watcher:", err)
		return err
	}

	watcher.SetHandler(FileEventHandler)

	for _, folder := range folders {
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			fmt.Println("Folder does not exist, skipping:", folder)
			continue
		}
		if err := watcher.Add(folder); err != nil {
			fmt.Println("Error adding folder to watcher:", err)
		}
	}

	watcher.Start()

	stopChan = make(chan any)
	go func() {
		<-stopChan
		watcher.Close()
	}()

	return nil
}

func StopWatcher() {
	if stopChan != nil {
		close(stopChan)
	}
}
