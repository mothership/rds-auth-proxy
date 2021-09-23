package file

import (
	"os"
	"strings"
)

func ExpandPath(filePath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(filePath, "$HOME", home), nil
}

func Exists(filePath string) bool {
	path, err := ExpandPath(filePath)
	if err != nil {
		// Actually panic here, because no homedir is ???
		panic(err)
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func DirExists(filePath string) bool {
	path, err := ExpandPath(filePath)
	if err != nil {
		// Actually panic here, because no homedir is ???
		panic(err)
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
