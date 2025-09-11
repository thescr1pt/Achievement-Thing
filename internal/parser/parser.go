package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

type Achievement struct {
	Name     string
	Achieved bool
}

func ParseFile(reader io.Reader, filename string) (map[string]Achievement, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".ini":
		return parseINI(reader)
	case ".json":
		return parseJSON(reader)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
}

func shouldIncludeAchievement(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}

	if strings.ToLower(name) == "steamachievements" {
		return false
	}

	return true
}

func parseINI(reader io.Reader) (map[string]Achievement, error) {
	achievements := make(map[string]Achievement)
	scanner := bufio.NewScanner(reader)

	var currentSection string
	var currentAchievement Achievement

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if currentSection != "" && shouldIncludeAchievement(currentSection) {
				achievements[currentSection] = currentAchievement
			}

			currentSection = strings.Trim(line, "[]")
			currentAchievement = Achievement{Name: currentSection}
			continue
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.ToLower(strings.TrimSpace(parts[0]))
			value := strings.TrimSpace(parts[1])

			if key == "achieved" || key == "state" || key == "haveachieved" || key == "unlocked" || key == "earned" {
				achieved, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("invalid boolean value for '%s' in section %s: %v", key, currentSection, err)
				}
				currentAchievement.Achieved = achieved
			}
		}
	}

	if currentSection != "" && shouldIncludeAchievement(currentSection) {
		achievements[currentSection] = currentAchievement
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading INI file: %v", err)
	}

	return achievements, nil
}

func parseJSON(reader io.Reader) (map[string]Achievement, error) {
	var data map[string]Achievement
	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return data, nil
}
