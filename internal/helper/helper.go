package helper

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func FindFilesRecursive(folder string, matchFiles []string) ([]string, error) {
	var results []string
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			lowerPath := strings.ToLower(path)
			for _, f := range matchFiles {
				if strings.HasSuffix(lowerPath, strings.ToLower(f)) {
					results = append(results, path)
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func ExtractAppId(filePath string) string {
	sep := string(os.PathSeparator)
	parts := strings.Split(filePath, sep)
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err == nil {
			return p
		}
	}
	return ""
}
