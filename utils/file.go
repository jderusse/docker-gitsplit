package utils

import (
	"github.com/otiai10/copy"
	"os"
	"path/filepath"
	"strings"
)

func FileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

func Copy(source string, target string) error {
	return copy.Copy(source, target)
}

func ResolvePath(path string) string {
	path = os.ExpandEnv(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		path = strings.Replace(path, "~", os.Getenv("HOME"), 1)
	}

	if filepath.IsAbs(path) {
		return path
	}

	pwd, err := os.Getwd()
	if err != nil {
		return path
	}

	return filepath.Join(pwd, path)
}
