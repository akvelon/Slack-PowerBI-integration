package utils

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// GetUniqueFileName build a unique filename using current time and random number
func GetUniqueFileName(fileOrigName, ext string) string {
	return fmt.Sprintf("%d_%s_%s.%s", rand.Intn(100), fileOrigName, time.Now().Format("2006.01.02_15.04.05.0000"), ext)
}

// GetAbsolutePath returns absolute path based on relative path
func GetAbsolutePath(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

// ReadFile return file content in string
func ReadFile(fileFullPath string) (string, error) {
	body, err := ioutil.ReadFile(fileFullPath)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// CheckAndCreateDir method for check directory on existence and create if
func CheckAndCreateDir(dir string) error {
	if !exists(dir) {
		return os.MkdirAll(dir, os.ModePerm)
	}

	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}
